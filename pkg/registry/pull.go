package registry

import (
	"context"
	"fmt"
	"io"
	"sync"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/containerd/containerd/images"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Pull(image, dir string, verbose bool, writer io.Writer) (*ocispec.Descriptor, error) {
	ctx := context.Background()
	cli, err := auth.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to get authenticating client to registry")
	}
	resolver, err := cli.Resolver(ctx)
	pullOpts := []oras.PullOpt{}

	fileStore := content.NewFileStore(dir)
	defer fileStore.Close()
	allowedMediaTypes := AllMimeTypes()

	if verbose {
		pullOpts = append(pullOpts, oras.WithPullBaseHandler(pullStatusTrack(writer)))
	}
	// pull the images
	desc, _, err := oras.Pull(ctx, resolver, image, fileStore, oras.WithAllowedMediaTypes(allowedMediaTypes))
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
