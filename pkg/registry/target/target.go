package target

import (
	"context"

	"github.com/containerd/containerd/remotes"
)

type Target interface {
	Resolver(ctx context.Context) (remotes.Resolver, error)
}
