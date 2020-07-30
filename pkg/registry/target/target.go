package target

import (
	"context"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
)

type Target interface {
	Resolver(ctx context.Context) (context.Context, ecresolver.ResolverCloser, error)
}
