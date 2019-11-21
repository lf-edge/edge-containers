package registry_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/oras/pkg/oras"
	"github.com/lf-edge/edge-containers/pkg/registry"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	testImageName = "docker.io/foo/testImage:abc"
	rootDiskType  = registry.Raw
	diskOneType   = registry.Qcow2
	configFile    = "config.json"
)

var (
	desc = ocispec.Descriptor{Digest: "sha256:abcdef123456"}
)

// MockedPush mocks calling oras.Push
type MockedPush struct {
	mock.Mock
}

func (m *MockedPush) Push(ctx context.Context, resolver remotes.Resolver, ref string, provider ctrcontent.Provider, descriptors []ocispec.Descriptor, opts ...oras.PushOpt) (ocispec.Descriptor, error) {
	m.Called(ctx, resolver, ref, provider, descriptors, opts)
	return desc, nil
}

type TestInputFile struct {
	name   string
	tmpdir string
}

func (t TestInputFile) fullname() string {
	return filepath.Join(t.tmpdir, t.name)
}
func (t TestInputFile) contents() []byte {
	return []byte(t.name)
}
func (t TestInputFile) size() int64 {
	return int64(len(t.contents()))
}
func (t TestInputFile) digest() digest.Digest {
	return digest.Digest(fmt.Sprintf("sha256:%x", sha256.Sum256(t.contents())))
}

func TestPush(t *testing.T) {
	// create a temporary directory and install basic test files
	tmpdir, err := ioutil.TempDir("", "eci-test")
	if err != nil {
		t.Fatalf("unable to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	// full paths
	inputs := map[string]TestInputFile{}
	inputs["kernel"] = TestInputFile{name: "kernel", tmpdir: tmpdir}
	inputs["initrd"] = TestInputFile{name: "initrd", tmpdir: tmpdir}
	inputs["root"] = TestInputFile{name: "root.raw", tmpdir: tmpdir}
	inputs["disk1"] = TestInputFile{name: "disk1.qcow2", tmpdir: tmpdir}
	// fill the files
	for _, v := range inputs {
		err = ioutil.WriteFile(v.fullname(), v.contents(), 0644)
		if err != nil {
			t.Fatalf("unable to create %s: %v", v.fullname(), err)
		}
	}
	validArtifact := &registry.Artifact{
		Config: inputs["config"].fullname(),
		Kernel: inputs["kernel"].fullname(),
		Initrd: inputs["initrd"].fullname(),
		Root:   &registry.Disk{Path: inputs["root"].fullname(), Type: rootDiskType},
		Disks:  []*registry.Disk{{Path: inputs["disk1"].fullname(), Type: diskOneType}},
	}
	// expected descriptors to be returned in normal mode
	expectedDescriptors := []ocispec.Descriptor{
		{MediaType: registry.MimeTypeECIKernel, Digest: inputs["kernel"].digest(), Size: inputs["kernel"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIKernel, registry.AnnotationRole: registry.RoleKernel, ocispec.AnnotationTitle: "kernel"}},
		{MediaType: registry.MimeTypeECIInitrd, Digest: inputs["initrd"].digest(), Size: inputs["initrd"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIInitrd, registry.AnnotationRole: registry.RoleInitrd, ocispec.AnnotationTitle: "initrd"}},
		{MediaType: registry.MimeTypeECIDiskRaw, Digest: inputs["root"].digest(), Size: inputs["root"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskRaw, registry.AnnotationRole: registry.RoleRootDisk, ocispec.AnnotationTitle: "disk-root-" + inputs["root"].name}},
		{MediaType: registry.MimeTypeECIDiskQcow2, Digest: inputs["disk1"].digest(), Size: inputs["disk1"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskQcow2, registry.AnnotationRole: registry.RoleAdditionalDisk, ocispec.AnnotationTitle: "disk-0-" + inputs["disk1"].name}},
	}
	// expected descriptors to be returned in lgeacy mode
	expectedDescriptorsLegacy := []ocispec.Descriptor{
		{MediaType: registry.MimeTypeOCIImageLayer, Digest: inputs["kernel"].digest(), Size: inputs["kernel"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIKernel, registry.AnnotationRole: registry.RoleKernel, ocispec.AnnotationTitle: "kernel"}},
		{MediaType: registry.MimeTypeOCIImageLayer, Digest: inputs["initrd"].digest(), Size: inputs["initrd"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIInitrd, registry.AnnotationRole: registry.RoleInitrd, ocispec.AnnotationTitle: "initrd"}},
		{MediaType: registry.MimeTypeOCIImageLayer, Digest: inputs["root"].digest(), Size: inputs["root"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskRaw, registry.AnnotationRole: registry.RoleRootDisk, ocispec.AnnotationTitle: "disk-root-" + inputs["root"].name}},
		{MediaType: registry.MimeTypeOCIImageLayer, Digest: inputs["disk1"].digest(), Size: inputs["disk1"].size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskQcow2, registry.AnnotationRole: registry.RoleAdditionalDisk, ocispec.AnnotationTitle: "disk-0-" + inputs["disk1"].name}},
	}

	tests := []struct {
		artifact *registry.Artifact
		image    string
		legacy   bool
		contents []ocispec.Descriptor
		digest   string
		opts     []oras.PushOpt
		err      error
	}{
		// no artifact
		{nil, testImageName, false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("must have valid Artifact")},
		// no image name
		{&registry.Artifact{}, "", false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("must have valid image ref")},
		// missing kernel file
		{&registry.Artifact{Kernel: "abcd.kernel"}, testImageName, false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("error adding kernel")},
		// missing initrd file
		{&registry.Artifact{Initrd: "abcd.initrd"}, testImageName, false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("error adding initrd")},
		// missing config file
		{&registry.Artifact{Config: "abcd.config"}, testImageName, false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("error adding config")},
		// missing root disk
		{&registry.Artifact{Root: &registry.Disk{Path: "abcd.diskroot", Type: rootDiskType}}, testImageName, false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("error adding disk-root")},
		// missing additional disk
		{&registry.Artifact{Disks: []*registry.Disk{{Path: "abcd.diskone", Type: registry.Vmdk}}}, testImageName, false, []ocispec.Descriptor{}, "", nil, fmt.Errorf("error adding disk-0")},
		// normal without legacy
		{validArtifact, testImageName, false, expectedDescriptors, string(desc.Digest), nil, nil},
		// normal with legacy
		{validArtifact, testImageName, true, expectedDescriptorsLegacy, string(desc.Digest), nil, nil},
	}
	for i, tt := range tests {
		// ensure it is called in the right way - this will check the arguments
		m := new(MockedPush)
		m.On("Push", mock.Anything, mock.Anything, tt.image, mock.Anything, tt.contents, tt.opts).Return(desc, nil)
		// create the Pusher
		pusher := registry.Pusher{
			Artifact: tt.artifact,
			Image:    tt.image,
			Impl:     m.Push,
		}
		dig, err := pusher.Push(tt.legacy, false, nil)
		switch {
		case (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("%d: mismatched errors, actual %v expected %v", i, err, tt.err)
		case err != nil:
			continue
		case dig != tt.digest:
			t.Errorf("%d: mismatched names, actual '%s', expected '%s'", i, dig, tt.digest)
		}
		// check that everything was called
		m.AssertExpectations(t)
	}
}
