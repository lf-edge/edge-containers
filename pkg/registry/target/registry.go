package target

import (
	"context"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
)

// Registry push to and pull from whichever registry is indicated by the image/
// For example, docker.io/library/alpine goes to docker hub.
type Registry struct {
}

func (r *Registry) Resolver(ctx context.Context) (ecresolver.ResolverCloser, error) {
	return ecresolver.NewRegistry(ctx)
}
