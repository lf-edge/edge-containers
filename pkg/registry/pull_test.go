package registry_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/lf-edge/edge-containers/pkg/registry"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
	"oras.land/oras-go/pkg/target"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// MockedPull mocks calling oras.Pull
type MockedPull struct {
	mock.Mock
}

func (m *MockedPull) Pull(ctx context.Context, from target.Target, fromRef string, to target.Target, toRef string, opts ...oras.CopyOpt) (ocispec.Descriptor, error) {
	m.Called(ctx, from, fromRef, to, opts)
	return desc, nil
}

func TestPull(t *testing.T) {
	tests := []struct {
		image  string
		digest string
		opts   []oras.CopyOpt
		err    error
	}{
		// no image name
		{"", "", nil, fmt.Errorf("must have valid image ref")},
		// normal
		{testImageName, string(desc.Digest), nil, nil},
	}
	for i, tt := range tests {
		// ensure it is called in the right way - this will check the arguments
		m := new(MockedPull)
		m.On("Pull", mock.Anything, mock.Anything, tt.image, mock.Anything, mock.Anything).Return(desc, nil, nil)
		// create the Puller
		puller := registry.Puller{
			Image: tt.image,
			Impl:  m.Pull,
		}
		_, resolver, err := ecresolver.NewRegistry(context.TODO())
		if err != nil {
			t.Errorf("unexpected error when created NewRegistry resolver: %v", err)
		}
		dig, _, err := puller.Pull(content.NewFile("/tmp/foo"), 0, false, nil, resolver)
		switch {
		case (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("%d: mismatched errors, actual %v expected %v", i, err, tt.err)
		case err != nil:
			continue
		case string(dig.Digest) != tt.digest:
			t.Errorf("%d: mismatched names, actual '%s', expected '%s'", i, string(dig.Digest), tt.digest)
		}
		// check that everything was called
		m.AssertExpectations(t)
	}
}
