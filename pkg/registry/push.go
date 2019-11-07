package registry

import (
	"context"
	"fmt"
	"io"
	"path"
	"sync"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Push(image string, artifact *Artifact, verbose bool, writer io.Writer) (string, error) {
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

	if artifact.Kernel != "" {
		role = RoleKernel
		name = "kernel"
		customMediaType = MimeTypeECIKernel
		filepath = artifact.Kernel
		mediaType = GetLayerMediaType(customMediaType, artifact.Legacy)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
	}

	if artifact.Initrd != "" {
		role = RoleInitrd
		name = "initrd"
		customMediaType = MimeTypeECIInitrd
		filepath = artifact.Initrd
		mediaType = GetLayerMediaType(customMediaType, artifact.Legacy)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
	}

	if disk := artifact.Root; disk != nil {
		role = RoleRootDisk
		customMediaType = TypeToMime[disk.Type]
		filepath = disk.Path
		name := fmt.Sprintf("disk-root-%s", path.Base(filepath))
		mediaType = GetLayerMediaType(customMediaType, artifact.Legacy)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
	}
	for i, disk := range artifact.Disks {
		if disk != nil {
			name := fmt.Sprintf("disk-%d-%s", i, path.Base(filepath))
			role = RoleAdditionalDisk
			customMediaType = TypeToMime[disk.Type]
			filepath = disk.Path
			mediaType = GetLayerMediaType(customMediaType, artifact.Legacy)
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
	if artifact.Config != "" {
		name = "config.json"
		customMediaType = MimeTypeECIConfig
		filepath = artifact.Config
		mediaType = GetConfigMediaType(customMediaType, artifact.Legacy)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return "", fmt.Errorf("error adding %s config at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		pushOpts = append(pushOpts, oras.WithConfig(desc))
	}

	// push the data
	desc, err = oras.Push(ctx, resolver, image, fileStore, pushContents, pushOpts...)
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
