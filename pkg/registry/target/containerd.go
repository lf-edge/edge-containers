package target

import (
	"context"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
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

func (d *Containerd) Resolver(ctx context.Context) (context.Context, ecresolver.ResolverCloser, error) {
	return ecresolver.NewContainerd(ctx, d.address, d.namespace)
}
