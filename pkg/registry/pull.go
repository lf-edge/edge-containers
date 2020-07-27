package registry

import (
	"context"
	"fmt"
	"io"
	"sync"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"

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
// The target overrides where to get the image by providing a local directory.
// For example, the image docker.io/library/alpine normally would be pulled from docker hub.
// If you provide a target of /tmp/foo, it will look in /tmp/foo for the blobs and manifest.
//
// The format for the target parameters is expected to change rapidly.
func (p *Puller) Pull(dir string, verbose bool, writer io.Writer, target string) (*ocispec.Descriptor, error) {
	// must have valid image ref
	if p.Image == "" {
		return nil, fmt.Errorf("must have valid image ref")
	}
	// ensure we have a real puller
	if p.Impl == nil {
		p.Impl = oras.Pull
	}

	ctx := context.Background()
	var (
		err      error
		resolver remotes.Resolver
	)
	if target != "" {
		resolver, err = ecresolver.NewDirectory(target)
		if err != nil {
			return nil, err
		}
	} else {
		cli, err := auth.NewClient()
		if err != nil {
			return nil, fmt.Errorf("unable to get authenticating client to registry: %v", err)
		}
		resolver, err = cli.Resolver(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to get resolver for registry: %v", err)
		}
	}
	pullOpts := []oras.PullOpt{}

	fileStore := content.NewFileStore(dir)
	defer fileStore.Close()
	allowedMediaTypes := AllMimeTypes()

	if verbose {
		pullOpts = append(pullOpts, oras.WithPullBaseHandler(pullStatusTrack(writer)))
	}
	// pull the images
	desc, _, err := p.Impl(ctx, resolver, p.Image, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
	if err != nil {
		return nil, err
	}
	return &desc, nil
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
