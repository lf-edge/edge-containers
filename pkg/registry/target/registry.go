package target

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/remotes"
	auth "github.com/deislabs/oras/pkg/auth/docker"
)

// Registry push to and pull from whichever registry is indicated by the image/
// For example, docker.io/library/alpine goes to docker hub.
type Registry struct {
}

func (r Registry) Resolver(ctx context.Context) (remotes.Resolver, error) {
	cli, err := auth.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to get authenticating client to registry: %v", err)
	}
	resolver, err := cli.Resolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get resolver for registry: %v", err)
	}
	return resolver, nil
}
