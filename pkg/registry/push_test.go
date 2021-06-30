package registry_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	ctrcontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/lf-edge/edge-containers/pkg/registry"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	"oras.land/oras-go/pkg/oras"

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
	desc     = ocispec.Descriptor{Digest: "sha256:abcdef123456"}
	initTime = time.Now()
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
	name             string
	processedName    string
	tmpdir           string
	fullname         string
	contents         []byte
	compressed       []byte
	digest           digest.Digest
	compressedDigest digest.Digest
}

func NewTestInputFile(name, processedName, tmpdir string) TestInputFile {
	t := TestInputFile{
		name:          name,
		processedName: processedName,
		tmpdir:        tmpdir,
		fullname:      filepath.Join(tmpdir, name),
		contents:      []byte(name),
	}
	out, _ := compress([]byte(name), processedName, initTime)
	t.compressed = out
	t.digest = digest.Digest(fmt.Sprintf("sha256:%x", sha256.Sum256(t.contents)))
	t.compressedDigest = digest.Digest(fmt.Sprintf("sha256:%x", sha256.Sum256(t.compressed)))
	return t
}

func (t TestInputFile) Fullname() string {
	return t.fullname
}
func (t TestInputFile) Contents() []byte {
	return t.contents
}

// legacyContents create the tgz that would get created
func (t TestInputFile) LegacyContents() []byte {
	return t.compressed
}
func (t TestInputFile) Size() int64 {
	return int64(len(t.contents))
}
func (t TestInputFile) LegacySize() int64 {
	return int64(len(t.compressed))
}
func (t TestInputFile) Digest() digest.Digest {
	return t.digest
}
func (t TestInputFile) LegacyDigest() digest.Digest {
	return t.compressedDigest
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
	inputs["kernel"] = NewTestInputFile("kernel", "kernel", tmpdir)
	inputs["initrd"] = NewTestInputFile("initrd", "initrd", tmpdir)
	inputs["root"] = NewTestInputFile("root.raw", "disk-root-root.raw", tmpdir)
	inputs["disk1"] = NewTestInputFile("disk1.qcow2", "disk-0-disk1.qcow2", tmpdir)
	// fill the files
	for _, v := range inputs {
		err = ioutil.WriteFile(v.Fullname(), v.Contents(), 0644)
		if err != nil {
			t.Fatalf("unable to create %s: %v", v.Fullname(), err)
		}
	}
	validArtifact := &registry.Artifact{
		Kernel: &registry.FileSource{Path: inputs["kernel"].Fullname()},
		Initrd: &registry.FileSource{Path: inputs["initrd"].Fullname()},
		Root:   &registry.Disk{Source: &registry.FileSource{Path: inputs["root"].Fullname()}, Type: rootDiskType},
		Disks:  []*registry.Disk{{Source: &registry.FileSource{Path: inputs["disk1"].Fullname()}, Type: diskOneType}},
	}
	// expected descriptors to be returned in normal mode
	expectedDescriptors := []ocispec.Descriptor{
		{MediaType: registry.MimeTypeECIKernel, Digest: inputs["kernel"].Digest(), Size: inputs["kernel"].Size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIKernel, registry.AnnotationRole: registry.RoleKernel, ocispec.AnnotationTitle: "kernel"}},
		{MediaType: registry.MimeTypeECIInitrd, Digest: inputs["initrd"].Digest(), Size: inputs["initrd"].Size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIInitrd, registry.AnnotationRole: registry.RoleInitrd, ocispec.AnnotationTitle: "initrd"}},
		{MediaType: registry.MimeTypeECIDiskRaw, Digest: inputs["root"].Digest(), Size: inputs["root"].Size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskRaw, registry.AnnotationRole: registry.RoleRootDisk, ocispec.AnnotationTitle: "disk-root-" + inputs["root"].name}},
		{MediaType: registry.MimeTypeECIDiskQcow2, Digest: inputs["disk1"].Digest(), Size: inputs["disk1"].Size(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskQcow2, registry.AnnotationRole: registry.RoleAdditionalDisk, ocispec.AnnotationTitle: "disk-0-" + inputs["disk1"].name}},
	}
	// expected descriptors to be returned in legacy mode
	expectedDescriptorsLegacy := []ocispec.Descriptor{
		{MediaType: registry.MimeTypeOCIImageLayerGzip, Digest: inputs["kernel"].LegacyDigest(), Size: inputs["kernel"].LegacySize(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIKernel, registry.AnnotationRole: registry.RoleKernel, ocispec.AnnotationTitle: "kernel"}},
		{MediaType: registry.MimeTypeOCIImageLayerGzip, Digest: inputs["initrd"].LegacyDigest(), Size: inputs["initrd"].LegacySize(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIInitrd, registry.AnnotationRole: registry.RoleInitrd, ocispec.AnnotationTitle: "initrd"}},
		{MediaType: registry.MimeTypeOCIImageLayerGzip, Digest: inputs["root"].LegacyDigest(), Size: inputs["root"].LegacySize(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskRaw, registry.AnnotationRole: registry.RoleRootDisk, ocispec.AnnotationTitle: "disk-root-" + inputs["root"].name}},
		{MediaType: registry.MimeTypeOCIImageLayerGzip, Digest: inputs["disk1"].LegacyDigest(), Size: inputs["disk1"].LegacySize(), Annotations: map[string]string{registry.AnnotationMediaType: registry.MimeTypeECIDiskQcow2, registry.AnnotationRole: registry.RoleAdditionalDisk, ocispec.AnnotationTitle: "disk-0-" + inputs["disk1"].name}},
	}

	tests := []struct {
		artifact *registry.Artifact
		image    string
		format   registry.Format
		contents []ocispec.Descriptor
		digest   string
		opts     []oras.PushOpt
		err      error
	}{
		// no artifact
		{nil, testImageName, registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("must have valid Artifact")},
		// no image name
		{&registry.Artifact{}, "", registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("must have valid image ref")},
		// missing kernel file
		{&registry.Artifact{Kernel: &registry.FileSource{Path: "abcd.kernel"}}, testImageName, registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("could not build manifest: error adding kernel")},
		// missing initrd file
		{&registry.Artifact{Initrd: &registry.FileSource{Path: "abcd.initrd"}}, testImageName, registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("could not build manifest: error adding initrd")},
		// missing config file
		{&registry.Artifact{Config: &registry.FileSource{Path: "abcd.config"}}, testImageName, registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("could not build manifest: error adding config")},
		// missing root disk
		{&registry.Artifact{Root: &registry.Disk{Source: &registry.FileSource{Path: "abcd.diskroot"}, Type: rootDiskType}}, testImageName, registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("could not build manifest: error adding disk-root")},
		// missing additional disk
		{&registry.Artifact{Disks: []*registry.Disk{{Source: &registry.FileSource{Path: "abcd.diskone"}, Type: registry.Vmdk}}}, testImageName, registry.FormatArtifacts, []ocispec.Descriptor{}, "", nil, fmt.Errorf("could not build manifest: error adding disk-0")},
		// normal without legacy
		{validArtifact, testImageName, registry.FormatArtifacts, expectedDescriptors, string(desc.Digest), nil, nil},
		// normal with legacy
		{validArtifact, testImageName, registry.FormatLegacy, expectedDescriptorsLegacy, string(desc.Digest), nil, nil},
	}
	for i, tt := range tests {
		// ensure it is called in the right way - this will check the arguments
		m := new(MockedPush)
		// TODO: the last argument here should check that the config is created
		m.On("Push", mock.Anything, mock.Anything, tt.image, mock.Anything, tt.contents, mock.MatchedBy(func(opts []oras.PushOpt) bool { return len(opts) == 1 })).Return(desc, nil)
		// create the Pusher
		pusher := registry.Pusher{
			Artifact:  tt.artifact,
			Image:     tt.image,
			Timestamp: &initTime,
			Impl:      m.Push,
		}
		_, resolver, err := ecresolver.NewRegistry(nil)
		if err != nil {
			t.Errorf("unexpected error when created NewRegistry resolver: %v", err)
		}
		dig, err := pusher.Push(tt.format, false, nil, registry.ConfigOpts{}, resolver)
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

func compress(in []byte, name string, timestamp time.Time) (out []byte, err error) {
	byteWriter := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(byteWriter)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	if err := addBytesToTarWriter(in, name, tarWriter, timestamp); err != nil {
		return nil, err
	}
	tarWriter.Close()
	gzipWriter.Close()
	out = byteWriter.Bytes()
	return
}

func addBytesToTarWriter(b []byte, name string, tw *tar.Writer, timestamp time.Time) error {
	// create the header
	header := &tar.Header{
		Name:    name,
		Size:    int64(len(b)),
		Mode:    0644,
		ModTime: timestamp,
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header: %v", err)
	}
	_, err := tw.Write(b)
	return err
}
