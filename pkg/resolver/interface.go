package resolver

import (
	"github.com/containerd/containerd/remotes"
)

type ResolverCloser interface {
	remotes.Resolver
	Finalize() error
}
