package target

import (
	"context"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"

	"github.com/containerd/containerd/remotes"
)

// Containerd push to and pull from containerd
type Containerd struct {
	address   string
	namespace string
}

func NewContainerd(address, namespace string) *Containerd {
	return &Containerd{
		address:   address,
		namespace: namespace,
	}
}

func (d *Containerd) Resolver(ctx context.Context) (remotes.Resolver, error) {
	return ecresolver.NewContainerd(d.address, d.namespace)
}
