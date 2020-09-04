package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	"github.com/lf-edge/edge-containers/pkg/store"
	"github.com/lf-edge/edge-containers/pkg/tgz"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/images"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	DefaultAuthor = "lf-edge/edge-containers"
	DefaultOS     = runtime.GOOS
	DefaultArch   = runtime.GOARCH
)

type Pusher struct {
	// Artifact artifact to push
	Artifact *Artifact
	// Image reference to image, e.g. docker.io/foo/bar:tagabc
	Image string
	// Timestamp set any files to have this timestamp, instead of the default of the file time
	Timestamp *time.Time
	// Impl the OCI artifacts pusher. Normally should be left blank, will be filled in to use oras. Override only for special cases like testing.
	Impl func(ctx context.Context, resolver remotes.Resolver, ref string, provider ctrcontent.Provider, descriptors []ocispec.Descriptor, opts ...oras.PushOpt) (ocispec.Descriptor, error)
}

// Push push the artifact to the appropriate registry. Arguments are the format to write,
// an io.Writer for sending debug output, ConfigOpts to configure how the image should be configured,
// and a target.
//
// The target determines the target type. target.Registry just uses the default registry,
// while target.Directory uses a local directory.
func (p Pusher) Push(format Format, verbose bool, writer io.Writer, configOpts ConfigOpts, resolver ecresolver.ResolverCloser) (string, error) {
	var (
		desc            ocispec.Descriptor
		mediaType       string
		customMediaType string
		role            string
		name            string
		filepath        string
		err             error
		pushOpts        []oras.PushOpt
	)

	// ensure the artifact and name are provided
	if p.Artifact == nil {
		return "", fmt.Errorf("must have valid Artifact")
	}
	if p.Image == "" {
		return "", fmt.Errorf("must have valid image ref")
	}
	// ensure we have a real pusher
	if p.Impl == nil {
		p.Impl = oras.Push
	}

	// get the saved context; if nil, create a background one
	ctx := resolver.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Go through each file type in the registry and add the appropriate file type and path, along with annotations
	fileStore := content.NewFileStore("")
	defer fileStore.Close()
	memStore := content.NewMemoryStore()
	multiStore := store.MultiReader{}
	multiStore.AddStore(fileStore, memStore)

	// if we have the container format, we need to create tgz layers
	var (
		tmpDir       string
		labels       = map[string]string{}
		pushContents = []ocispec.Descriptor{}
		layers       = []digest.Digest{}
		layerHash    digest.Digest
	)

	if format == FormatLegacy {
		tmpDir, err = ioutil.TempDir("", "edge-containers")
		if err != nil {
			return "", fmt.Errorf("could not make temporary directory for tgz files: %v", err)
		}
		defer os.RemoveAll(tmpDir)
	}

	if p.Artifact.Kernel != "" {
		role = RoleKernel
		name = "kernel"
		layerHash = ""
		customMediaType = MimeTypeECIKernel
		filepath = p.Artifact.Kernel
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatLegacy {
			tgzfile := path.Join(tmpDir, name)
			tarHash, _, err := tgz.Compress(filepath, name, tgzfile, p.Timestamp)
			if err != nil {
				return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
			// convert the tarHash into a digest
			layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
		if layerHash == "" {
			layerHash = desc.Digest
		}
		layers = append(layers, layerHash)

		labels[AnnotationKernelPath] = fmt.Sprintf("/%s", name)
	}

	if p.Artifact.Initrd != "" {
		role = RoleInitrd
		name = "initrd"
		layerHash = ""
		customMediaType = MimeTypeECIInitrd
		filepath = p.Artifact.Initrd
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatLegacy {
			tgzfile := path.Join(tmpDir, name)
			tarHash, _, err := tgz.Compress(filepath, name, tgzfile, p.Timestamp)
			if err != nil {
				return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
			layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
		if layerHash == "" {
			layerHash = desc.Digest
		}
		layers = append(layers, layerHash)

		labels[AnnotationInitrdPath] = fmt.Sprintf("/%s", name)
	}

	if disk := p.Artifact.Root; disk != nil {
		role = RoleRootDisk
		customMediaType = TypeToMime[disk.Type]
		filepath = disk.Path
		name := fmt.Sprintf("disk-root-%s", path.Base(filepath))
		layerHash = ""
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatLegacy {
			tgzfile := path.Join(tmpDir, name)
			tarHash, _, err := tgz.Compress(filepath, name, tgzfile, p.Timestamp)
			if err != nil {
				return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
			layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
		if layerHash == "" {
			layerHash = desc.Digest
		}
		layers = append(layers, layerHash)

		labels[AnnotationRootPath] = fmt.Sprintf("/%s", name)
	}
	for i, disk := range p.Artifact.Disks {
		if disk != nil {
			role = RoleAdditionalDisk
			customMediaType = TypeToMime[disk.Type]
			filepath = disk.Path
			name := fmt.Sprintf("disk-%d-%s", i, path.Base(filepath))
			layerHash = ""
			mediaType = GetLayerMediaType(customMediaType, format)
			if format == FormatLegacy {
				tgzfile := path.Join(tmpDir, name)
				tarHash, _, err := tgz.Compress(filepath, name, tgzfile, p.Timestamp)
				if err != nil {
					return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
				}
				filepath = tgzfile
				layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
			}
			desc, err = fileStore.Add(name, mediaType, filepath)
			if err != nil {
				return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
			}
			desc.Annotations[AnnotationMediaType] = customMediaType
			desc.Annotations[AnnotationRole] = role
			desc.Annotations[ocispec.AnnotationTitle] = name
			pushContents = append(pushContents, desc)
			if layerHash == "" {
				layerHash = desc.Digest
			}
			layers = append(layers, layerHash)

			labels[fmt.Sprintf(AnnotationDiskIndexPathPattern, i)] = fmt.Sprintf("/%s", name)
		}
	}

	// was a config specified?
	if p.Artifact.Config != "" {
		name = "config.json"
		customMediaType = MimeTypeECIConfig
		filepath = p.Artifact.Config
		mediaType = GetConfigMediaType(customMediaType, format)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s config at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
	} else {
		// for container format, we expect to have a specific config so docker can work with it
		created := time.Now()
		configAuthor, configOS, configArch := configOpts.Author, configOpts.OS, configOpts.Architecture
		if configAuthor == "" {
			configAuthor = DefaultAuthor
		}
		if configOS == "" {
			configOS = DefaultOS
		}
		if configArch == "" {
			configArch = DefaultArch
		}
		config := ocispec.Image{
			Created:      &created,
			Author:       configAuthor,
			Architecture: configArch,
			OS:           configOS,
			RootFS: ocispec.RootFS{
				Type:    "layers",
				DiffIDs: layers,
			},
			Config: ocispec.ImageConfig{
				Labels: labels,
			},
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("error marshaling config to json: %v", err)
		}

		name = "config.json"
		mediaType = MimeTypeOCIImageConfig
		desc = memStore.Add(name, mediaType, configBytes)
		if err != nil {
			return "", fmt.Errorf("error adding OCI config: %v", err)
		}
	}
	pushOpts = append(pushOpts, oras.WithConfig(desc))

	if verbose {
		pushOpts = append(pushOpts, oras.WithPushBaseHandler(pushStatusTrack(writer)))
	}

	// push the data
	desc, err = p.Impl(ctx, resolver, p.Image, multiStore, pushContents, pushOpts...)
	if err != nil {
		return "", err
	}
	if err := resolver.Finalize(ctx); err != nil {
		return desc.Digest.String(), fmt.Errorf("failed to finalize: %v", err)
	}
	return desc.Digest.String(), nil
}

func pushStatusTrack(writer io.Writer) images.Handler {
	var printLock sync.Mutex
	return images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if name, ok := content.ResolveName(desc); ok {
			printLock.Lock()
			defer printLock.Unlock()
			writer.Write([]byte(fmt.Sprintf("Uploading %s %s\n", desc.Digest.Encoded()[:12], name)))
		}
		return nil, nil
	})
}
