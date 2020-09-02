package registry

import (
	"context"
	"fmt"
	"io"
	"sync"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/lf-edge/edge-containers/pkg/registry/target"

	"github.com/containerd/containerd/images"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Puller struct {
	// Image reference to image, e.g. docker.io/foo/bar:tagabc
	Image string
	// Impl the OCI artifacts puller. Normally should be left blank, will be filled in to use oras. Override only for special cases like testing.
	Impl func(ctx context.Context, resolver remotes.Resolver, ref string, ingester ctrcontent.Ingester, opts ...oras.PullOpt) (ocispec.Descriptor, []ocispec.Descriptor, error)
}

// Pull pull the artifact from the appropriate registry and save it to a local directory.
// Arguments are the dir where to write it, an io.Writer for logging output, and a target.
//
// The target determines the target type. target.Registry just uses the default registry,
// while target.Directory uses a local directory, etc.
func (p *Puller) Pull(dir string, verbose bool, writer io.Writer, target target.Target) (*ocispec.Descriptor, *Artifact, error) {
	// must have valid image ref
	if p.Image == "" {
		return nil, nil, fmt.Errorf("must have valid image ref")
	}
	// ensure we have a real puller
	if p.Impl == nil {
		p.Impl = oras.Pull
	}

	var (
		err error
	)
	ctx, resolver, err := target.Resolver(context.Background())
	if err != nil {
		return nil, nil, err
	}
	pullOpts := []oras.PullOpt{}

	fileStore := content.NewFileStore(dir)
	defer fileStore.Close()
	allowedMediaTypes := AllMimeTypes()

	if verbose {
		pullOpts = append(pullOpts, oras.WithPullBaseHandler(pullStatusTrack(writer)))
	}
	// pull the images
	desc, layers, err := p.Impl(ctx, resolver, p.Image, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return nil, nil, err
	}
	// now process the layers to fill in our artifact
	artifact := &Artifact{
		Disks: []*Disk{},
	}
	for _, l := range layers {
		if l.Annotations == nil {
			continue
		}
		filepath := l.Annotations[ocispec.AnnotationTitle]
		if filepath == "" {
			continue
		}
		mediaType := l.Annotations[AnnotationMediaType]
		switch l.Annotations[AnnotationRole] {
		case RoleKernel:
			artifact.Kernel = filepath
		case RoleInitrd:
			artifact.Initrd = filepath
		case RoleRootDisk:
			artifact.Root = &Disk{
				Path: filepath,
				Type: MimeToType[mediaType],
			}
		case AnnotationMediaType:
			artifact.Disks = append(artifact.Disks, &Disk{
				Path: filepath,
				Type: MimeToType[mediaType],
			})
		}
	}
	return &desc, artifact, nil
}

func pullStatusTrack(writer io.Writer) images.Handler {
	var printLock sync.Mutex
	return images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if name, ok := content.ResolveName(desc); ok {
			digestString := desc.Digest.String()
			if err := desc.Digest.Validate(); err == nil {
				if algo := desc.Digest.Algorithm(); algo == digest.SHA256 {
					digestString = desc.Digest.Encoded()[:12]
				}
			}
			printLock.Lock()
			defer printLock.Unlock()
			writer.Write([]byte(fmt.Sprintf("Downloaded %s %s\n", digestString, name)))
		}
		return nil, nil
	})
}
