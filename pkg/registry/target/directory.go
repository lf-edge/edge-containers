package target

import (
	"context"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"

	"github.com/containerd/containerd/remotes"
)

// Directory push to and pull from a local directory.
type Directory struct {
	dir string
}

func NewDirectory(dir string) *Directory {
	return &Directory{
		dir: dir,
	}
}

func (d *Directory) Resolver(ctx context.Context) (remotes.Resolver, error) {
	return ecresolver.NewDirectory(d.dir)
}
