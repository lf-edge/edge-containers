package target

import (
	"context"

	"github.com/containerd/containerd/remotes"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
)

// Resolver push to and pull from a remotes.Resolver
type Resolver struct {
	resolver remotes.Resolver
}

func NewResolver(resolver remotes.Resolver) *Resolver {
	return &Resolver{
		resolver: resolver,
	}
}

func (d *Resolver) Resolver(ctx context.Context) (context.Context, ecresolver.ResolverCloser, error) {
	return ecresolver.NewResolver(ctx, d.resolver)
}
