package resolver

import (
	"context"
	"testing"

	"oras.land/oras-go/pkg/auth"
)

// TestRegistryOptResolverOptions checks that each RegistryOpt reaches the oras
// resolver setting it stands for. A constructed Registry keeps no record of the
// settings it was built from, so this translation is only observable here.
func TestRegistryOptResolverOptions(t *testing.T) {
	tests := []struct {
		name      string
		opts      []RegistryOpt
		plainHTTP bool
	}{
		{"default", nil, false},
		{"plain HTTP", []RegistryOpt{WithPlainHTTP()}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var settings registryOpts
			for _, opt := range tt.opts {
				opt(&settings)
			}
			var got auth.ResolverSettings
			for _, opt := range settings.resolverOptions() {
				opt(&got)
			}
			if got.PlainHTTP != tt.plainHTTP {
				t.Errorf("PlainHTTP = %v, want %v", got.PlainHTTP, tt.plainHTTP)
			}
		})
	}
}

// TestNewRegistryDefaults checks that NewRegistry stays equivalent to
// NewRegistryWithOpts with no options.
func TestNewRegistryDefaults(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		call func() (context.Context, *Registry, error)
	}{
		{"NewRegistry", func() (context.Context, *Registry, error) { return NewRegistry(ctx) }},
		{"NewRegistryWithOpts", func() (context.Context, *Registry, error) { return NewRegistryWithOpts(ctx) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gotCtx, reg, err := tt.call()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if reg.Resolver == nil {
				t.Error("no resolver returned")
			}
			if gotCtx != ctx || reg.Context() != ctx {
				t.Error("context not carried through")
			}
			if err := reg.Finalize(ctx); err != nil {
				t.Errorf("Finalize: %v", err)
			}
		})
	}
}
