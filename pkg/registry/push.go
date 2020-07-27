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

	"github.com/lf-edge/edge-containers/pkg/store"
	"github.com/lf-edge/edge-containers/pkg/tgz"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	auth "github.com/deislabs/oras/pkg/auth/docker"
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
	// Impl the OCI artifacts pusher. Normally should be left blank, will be filled in to use oras. Override only for special cases like testing.
	Impl func(ctx context.Context, resolver remotes.Resolver, ref string, provider ctrcontent.Provider, descriptors []ocispec.Descriptor, opts ...oras.PushOpt) (ocispec.Descriptor, error)
}

func (p Pusher) Push(format Format, verbose bool, writer io.Writer, configOpts ConfigOpts) (string, error) {
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

	ctx := context.Background()
	cli, err := auth.NewClient()
	if err != nil {
		return "", fmt.Errorf("unable to get authenticating client to registry")
	}
	resolver, err := cli.Resolver(ctx)

	// Go through each file type in the registry and add the appropriate file type and path, along with annotations
	fileStore := content.NewFileStore("")
	defer fileStore.Close()
	memStore := content.NewMemoryStore()
	multiStore := store.MultiReader{}
	multiStore.AddStore(fileStore, memStore)

	// if we have the container format, we need to create tgz layers
	var tmpDir string

	if format == FormatContainer {
		tmpDir, err = ioutil.TempDir("", "edge-containers")
		if err != nil {
			return "", fmt.Errorf("could not make temporary directory for tgz files: %v", err)
		}
		defer os.RemoveAll(tmpDir)
	}

	labels := map[string]string{}

	pushContents := []ocispec.Descriptor{}

	if p.Artifact.Kernel != "" {
		role = RoleKernel
		name = "kernel"
		customMediaType = MimeTypeECIKernel
		filepath = p.Artifact.Kernel
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatContainer {
			tgzfile := path.Join(tmpDir, name)
			err = tgz.Compress(filepath, name, tgzfile)
			if err != nil {
				return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)

		labels[AnnotationKernelPath] = fmt.Sprintf("/%s", name)
	}

	if p.Artifact.Initrd != "" {
		role = RoleInitrd
		name = "initrd"
		customMediaType = MimeTypeECIInitrd
		filepath = p.Artifact.Initrd
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatContainer {
			tgzfile := path.Join(tmpDir, name)
			err = tgz.Compress(filepath, name, tgzfile)
			if err != nil {
				return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)

		labels[AnnotationInitrdPath] = fmt.Sprintf("/%s", name)
	}

	if disk := p.Artifact.Root; disk != nil {
		role = RoleRootDisk
		customMediaType = TypeToMime[disk.Type]
		filepath = disk.Path
		name := fmt.Sprintf("disk-root-%s", path.Base(filepath))
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatContainer {
			tgzfile := path.Join(tmpDir, name)
			err = tgz.Compress(filepath, name, tgzfile)
			if err != nil {
				return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)

		labels[AnnotationRootPath] = fmt.Sprintf("/%s", name)
	}
	for i, disk := range p.Artifact.Disks {
		if disk != nil {
			role = RoleAdditionalDisk
			customMediaType = TypeToMime[disk.Type]
			filepath = disk.Path
			name := fmt.Sprintf("disk-%d-%s", i, path.Base(filepath))
			mediaType = GetLayerMediaType(customMediaType, format)
			if format == FormatContainer {
				tgzfile := path.Join(tmpDir, name)
				err = tgz.Compress(filepath, name, tgzfile)
				if err != nil {
					return "", fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
				}
				filepath = tgzfile
			}
			desc, err = fileStore.Add(name, mediaType, filepath)
			if err != nil {
				return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
			}
			desc.Annotations[AnnotationMediaType] = customMediaType
			desc.Annotations[AnnotationRole] = role
			desc.Annotations[ocispec.AnnotationTitle] = name
			pushContents = append(pushContents, desc)

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
		pushOpts = append(pushOpts, oras.WithConfig(desc))
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
				DiffIDs: []digest.Digest{},
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
		pushOpts = append(pushOpts, oras.WithConfig(desc))
	}

	if verbose {
		pushOpts = append(pushOpts, oras.WithPushBaseHandler(pushStatusTrack(writer)))
	}

	// push the data
	desc, err = p.Impl(ctx, resolver, p.Image, multiStore, pushContents, pushOpts...)
	if err != nil {
		return "", err
	}
	return string(desc.Digest), nil
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
