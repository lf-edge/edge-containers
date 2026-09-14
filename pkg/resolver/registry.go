package resolver

/*
 Provides a github.com/containerd/containerd/remotes#Resolver that resolves
 to an OCI registry, authenticating from the local docker credential store.

*/

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/remotes"
	"oras.land/oras-go/pkg/auth"
	authdocker "oras.land/oras-go/pkg/auth/docker"
)

type Registry struct {
	remotes.Resolver
	ctx context.Context
}

// registryOpts settings accumulated by the RegistryOpt passed to NewRegistryWithOpts.
type registryOpts struct {
	plainHTTP bool
}

// RegistryOpt configures a Registry created by NewRegistryWithOpts.
type RegistryOpt func(*registryOpts)

// WithPlainHTTP directs the resolver to contact the registry over HTTP rather than
// HTTPS, for a registry served without TLS such as a lab-local or test registry.
func WithPlainHTTP() RegistryOpt {
	return func(o *registryOpts) {
		o.plainHTTP = true
	}
}

// NewRegistry create a Registry resolver that reaches the registry over HTTPS.
func NewRegistry(ctx context.Context) (context.Context, *Registry, error) {
	return NewRegistryWithOpts(ctx)
}

// resolverOptions translate the settings into the equivalent oras resolver options.
func (o registryOpts) resolverOptions() []auth.ResolverOption {
	var opts []auth.ResolverOption
	if o.plainHTTP {
		opts = append(opts, auth.WithResolverPlainHTTP())
	}
	return opts
}

// NewRegistryWithOpts create a Registry resolver configured by opts.
func NewRegistryWithOpts(ctx context.Context, opts ...RegistryOpt) (context.Context, *Registry, error) {
	var settings registryOpts
	for _, opt := range opts {
		opt(&settings)
	}
	cli, err := authdocker.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get authenticating client to registry: %v", err)
	}
	resolver, err := cli.ResolverWithOpts(settings.resolverOptions()...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get resolver for registry: %v", err)
	}
	return ctx, &Registry{Resolver: resolver, ctx: ctx}, nil
}

func (r *Registry) Finalize(ctx context.Context) error {
	return nil
}

func (r *Registry) Context() context.Context {
	return r.ctx
}
