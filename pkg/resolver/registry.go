package resolver

/*
 Provides a github.com/containerd/containerd/remotes#Resolver that resolves
 to a containerd socket

*/

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/remotes"
	auth "github.com/deislabs/oras/pkg/auth/docker"
)

type Registry struct {
	remotes.Resolver
}

func NewRegistry(ctx context.Context) (*Registry, error) {
	cli, err := auth.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to get authenticating client to registry: %v", err)
	}
	resolver, err := cli.Resolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get resolver for registry: %v", err)
	}
	return &Registry{Resolver: resolver}, nil
}

func (r *Registry) Finalize() error {
	return nil
}
