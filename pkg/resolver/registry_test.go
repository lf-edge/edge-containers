package resolver

import (
	"context"
	"testing"

	"oras.land/oras-go/v2/registry/remote"
)

const testRef = "example.com/foo/bar:v1"

// TestRegistryTargetPlainHTTP checks that each RegistryOpt reaches the repository
// setting it stands for.
func TestRegistryTargetPlainHTTP(t *testing.T) {
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
			ctx := context.Background()
			_, reg, err := NewRegistryWithOpts(ctx, tt.opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			target, err := reg.Target(ctx, testRef)
			if err != nil {
				t.Fatalf("Target: %v", err)
			}
			repo, ok := target.(*remote.Repository)
			if !ok {
				t.Fatalf("Target returned %T, want *remote.Repository", target)
			}
			if repo.PlainHTTP != tt.plainHTTP {
				t.Errorf("PlainHTTP = %v, want %v", repo.PlainHTTP, tt.plainHTTP)
			}
			if repo.Client == nil {
				t.Error("repository has no client, so it would not authenticate")
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
			if gotCtx != ctx || reg.Context() != ctx {
				t.Error("context not carried through")
			}
			target, err := reg.Target(ctx, testRef)
			if err != nil {
				t.Fatalf("Target: %v", err)
			}
			if repo := target.(*remote.Repository); repo.PlainHTTP {
				t.Error("default reaches the registry over plain HTTP")
			}
			if err := reg.Finalize(ctx); err != nil {
				t.Errorf("Finalize: %v", err)
			}
		})
	}
}

// TestRegistryTargetRejectsBadReference checks that an unparseable reference is
// reported rather than producing a target that fails later.
func TestRegistryTargetRejectsBadReference(t *testing.T) {
	ctx := context.Background()
	_, reg, err := NewRegistry(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := reg.Target(ctx, "not a reference"); err == nil {
		t.Error("expected an error for an invalid reference")
	}
}

// TestRegistryTargetRejectsUppercaseRepository records that a registry reference
// must be lowercase, as the OCI distribution spec requires. containerd's parser
// accepted mixed case, so a reference that names a repository that way now fails
// where it once reached the registry and failed there instead.
func TestRegistryTargetRejectsUppercaseRepository(t *testing.T) {
	ctx := context.Background()
	_, reg, err := NewRegistry(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := reg.Target(ctx, "docker.io/foo/testImage:abc"); err == nil {
		t.Error("expected an error for an uppercase repository name")
	}
}
