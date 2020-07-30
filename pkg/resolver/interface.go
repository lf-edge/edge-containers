package resolver

import (
	"context"

	"github.com/containerd/containerd/remotes"
)

type ResolverCloser interface {
	remotes.Resolver
	Finalize(ctx context.Context) error
}
