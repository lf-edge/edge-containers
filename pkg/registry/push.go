package registry

import (
	"context"
	"fmt"
	"io"
	"path"
	"sync"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Pusher struct {
	// Artifact artifact to push
	Artifact *Artifact
	// Image reference to image, e.g. docker.io/foo/bar:tagabc
	Image string
	// Impl the OCI artifacts pusher. Normally should be left blank, will be filled in to use oras. Override only for special cases like testing.
	Impl func(ctx context.Context, resolver remotes.Resolver, ref string, provider ctrcontent.Provider, descriptors []ocispec.Descriptor, opts ...oras.PushOpt) (ocispec.Descriptor, error)
}

func (p Pusher) Push(format Format, verbose bool, writer io.Writer) (string, error) {
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

	pushContents := []ocispec.Descriptor{}

	if p.Artifact.Kernel != "" {
		role = RoleKernel
		name = "kernel"
		customMediaType = MimeTypeECIKernel
		filepath = p.Artifact.Kernel
		mediaType = GetLayerMediaType(customMediaType, format)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
	}

	if p.Artifact.Initrd != "" {
		role = RoleInitrd
		name = "initrd"
		customMediaType = MimeTypeECIInitrd
		filepath = p.Artifact.Initrd
		mediaType = GetLayerMediaType(customMediaType, format)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
	}

	if disk := p.Artifact.Root; disk != nil {
		role = RoleRootDisk
		customMediaType = TypeToMime[disk.Type]
		filepath = disk.Path
		name := fmt.Sprintf("disk-root-%s", path.Base(filepath))
		mediaType = GetLayerMediaType(customMediaType, format)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
	}
	for i, disk := range p.Artifact.Disks {
		if disk != nil {
			role = RoleAdditionalDisk
			customMediaType = TypeToMime[disk.Type]
			filepath = disk.Path
			name := fmt.Sprintf("disk-%d-%s", i, path.Base(filepath))
			mediaType = GetLayerMediaType(customMediaType, format)
			desc, err = fileStore.Add(name, mediaType, filepath)
			if err != nil {
				return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
			}
			desc.Annotations[AnnotationMediaType] = customMediaType
			desc.Annotations[AnnotationRole] = role
			desc.Annotations[ocispec.AnnotationTitle] = name
			pushContents = append(pushContents, desc)
		}
	}

	if verbose {
		pushOpts = append(pushOpts, oras.WithPushBaseHandler(pushStatusTrack(writer)))
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
	}

	// push the data
	desc, err = p.Impl(ctx, resolver, p.Image, fileStore, pushContents, pushOpts...)
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
