package registry_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/lf-edge/edge-containers/pkg/registry"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestManifest(t *testing.T) {
	// create a temporary directory and install basic test files
	tmpdir, err := os.MkdirTemp("", "eci-test")
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
		err = os.WriteFile(v.Fullname(), v.Contents(), 0644)
		if err != nil {
			t.Fatalf("unable to create %s: %v", v.Fullname(), err)
		}
	}
	validArtifact := &registry.Artifact{
		Config: nil,
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
		format   registry.Format
		contents []ocispec.Descriptor
		err      error
	}{
		// missing kernel file
		{&registry.Artifact{Kernel: &registry.FileSource{Path: "abcd.kernel"}}, registry.FormatArtifacts, []ocispec.Descriptor{}, fmt.Errorf("error adding kernel")},
		// missing initrd file
		{&registry.Artifact{Initrd: &registry.FileSource{Path: "abcd.initrd"}}, registry.FormatArtifacts, []ocispec.Descriptor{}, fmt.Errorf("error adding initrd")},
		// missing config file
		{&registry.Artifact{Config: &registry.FileSource{Path: "abcd.config"}}, registry.FormatArtifacts, []ocispec.Descriptor{}, fmt.Errorf("error adding config")},
		// missing root disk
		{&registry.Artifact{Root: &registry.Disk{Source: &registry.FileSource{Path: "abcd.diskroot"}, Type: rootDiskType}}, registry.FormatArtifacts, []ocispec.Descriptor{}, fmt.Errorf("error adding disk-root")},
		// missing additional disk
		{&registry.Artifact{Disks: []*registry.Disk{{Source: &registry.FileSource{Path: "abcd.diskone"}, Type: registry.Vmdk}}}, registry.FormatArtifacts, []ocispec.Descriptor{}, fmt.Errorf("error adding disk-0")},
		// normal without legacy
		{validArtifact, registry.FormatArtifacts, expectedDescriptors, nil},
		// normal with legacy
		{validArtifact, registry.FormatLegacy, expectedDescriptorsLegacy, nil},
	}
	for i, tt := range tests {
		var (
			manifestTmpDir string
			legacyOpts     []registry.LegacyOpt
		)
		legacyOpts = append(legacyOpts, registry.WithTimestamp(&initTime))

		if tt.format == registry.FormatLegacy {
			manifestTmpDir, err = os.MkdirTemp("", "edge-containers")
			if err != nil {
				t.Fatalf("could not make temporary directory for tgz files: %v", err)
			}
			legacyOpts = append(legacyOpts, registry.WithTmpDir(manifestTmpDir))
			defer os.RemoveAll(manifestTmpDir)
		}

		manifest, _, err := tt.artifact.Manifest(tt.format, registry.ConfigOpts{}, "", legacyOpts...)
		switch {
		case (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("%d: mismatched errors, actual %v expected %v", i, err, tt.err)
		case err != nil:
			continue
		case len(manifest.Layers) != len(tt.contents):
			t.Errorf("%d: mismatched layers length, actual %v, expected %v", i, manifest.Layers, tt.contents)
		default:
			for j, l := range manifest.Layers {
				if !equalLayer(l, tt.contents[j]) {
					t.Errorf("%d: %d: mismatched layer actual %v, expected %v", i, j, l, tt.contents[j])
				}
			}
		}
	}
}

func equalLayer(a, b ocispec.Descriptor) bool {
	return a.MediaType == b.MediaType && a.Size == b.Size && a.Digest == b.Digest &&
		equalMapStringString(a.Annotations, b.Annotations)
}

func equalMapStringString(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if v2, ok := b[k]; !ok || v != v2 {
			return false
		}
	}
	return true
}
