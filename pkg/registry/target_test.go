package registry_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"testing"

	"github.com/lf-edge/edge-containers/pkg/registry"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// TestFilesTargetVerifiesDigest covers the check on content arriving at a writer.
// A caller that sets AcceptHash takes the descriptor's word for the digest and
// gets the content either way, which is what a caller writing a disk image
// straight to a partition asks for.
func TestFilesTargetVerifiesDigest(t *testing.T) {
	payload := []byte("kernel content")
	descFor := func(d digest.Digest) ocispec.Descriptor {
		return ocispec.Descriptor{
			MediaType:   registry.MimeTypeECIKernel,
			Digest:      d,
			Size:        int64(len(payload)),
			Annotations: map[string]string{registry.AnnotationRole: registry.RoleKernel},
		}
	}

	tests := []struct {
		name       string
		desc       ocispec.Descriptor
		acceptHash bool
		wantErr    bool
	}{
		{"matching digest", descFor(digest.FromBytes(payload)), false, false},
		{"mismatched digest", descFor(digest.FromString("something else")), false, true},
		{"mismatched digest with AcceptHash", descFor(digest.FromString("something else")), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			target := &registry.FilesTarget{Kernel: &buf, AcceptHash: tt.acceptHash}
			err := target.Push(context.Background(), tt.desc, bytes.NewReader(payload))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Push error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !bytes.Equal(buf.Bytes(), payload) {
				t.Errorf("wrote %q, want %q", buf.Bytes(), payload)
			}
		})
	}
}

// TestFilesTargetDiscardsUnwantedRoles checks that content with no writer behind it
// is consumed rather than erroring, so the rest of the artifact still arrives.
func TestFilesTargetDiscardsUnwantedRoles(t *testing.T) {
	payload := []byte("initrd content")
	var kernel bytes.Buffer
	target := &registry.FilesTarget{Kernel: &kernel}
	desc := ocispec.Descriptor{
		MediaType:   registry.MimeTypeECIInitrd,
		Digest:      digest.FromBytes(payload),
		Size:        int64(len(payload)),
		Annotations: map[string]string{registry.AnnotationRole: registry.RoleInitrd},
	}
	if err := target.Push(context.Background(), desc, bytes.NewReader(payload)); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if kernel.Len() != 0 {
		t.Errorf("initrd content reached the kernel writer: %q", kernel.Bytes())
	}
}

// TestFilesTargetUnpacksTarLayers covers the tar layer media types, compressed and
// not. A registry is free to serve either, and the content the caller wants is
// inside the tar in both cases -- an unpacked-only-if-gzipped target hands the tar
// container to a writer expecting a kernel or a disk image.
func TestFilesTargetUnpacksTarLayers(t *testing.T) {
	payload := []byte("kernel content")
	tarred := tarOf(t, "kernel", payload)

	for _, tt := range []struct {
		name      string
		mediaType string
		blob      []byte
	}{
		{"uncompressed oci", registry.MimeTypeOCIImageLayer, tarred},
		{"uncompressed docker", registry.MimeTypeDockerLayerTar, tarred},
		{"gzipped oci", registry.MimeTypeOCIImageLayerGzip, gzipOf(t, tarred)},
		{"gzipped docker", registry.MimeTypeDockerLayerTarGzip, gzipOf(t, tarred)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var kernel bytes.Buffer
			target := &registry.FilesTarget{Kernel: &kernel}
			ctx := context.Background()

			// the config names the path inside the layer, which is what routes it
			config := ocispec.Image{}
			config.Config.Labels = map[string]string{registry.AnnotationKernelPath: "kernel"}
			configJSON, err := json.Marshal(config)
			if err != nil {
				t.Fatalf("marshal config: %v", err)
			}
			configDesc := ocispec.Descriptor{
				MediaType: registry.MimeTypeOCIImageConfig,
				Digest:    digest.FromBytes(configJSON),
				Size:      int64(len(configJSON)),
			}
			if err := target.Push(ctx, configDesc, bytes.NewReader(configJSON)); err != nil {
				t.Fatalf("push config: %v", err)
			}

			desc := ocispec.Descriptor{
				MediaType: tt.mediaType,
				Digest:    digest.FromBytes(tt.blob),
				Size:      int64(len(tt.blob)),
			}
			if err := target.Push(ctx, desc, bytes.NewReader(tt.blob)); err != nil {
				t.Fatalf("push layer: %v", err)
			}
			if !bytes.Equal(kernel.Bytes(), payload) {
				t.Errorf("kernel = %d bytes, want the %d-byte payload", kernel.Len(), len(payload))
			}
		})
	}
}

// TestFilesTargetRejectsDigestlessDescriptor checks that a descriptor carrying no
// digest is refused. Nothing can be verified against it, and the digest algorithm
// cannot be asked anything about it.
func TestFilesTargetRejectsDigestlessDescriptor(t *testing.T) {
	payload := []byte("kernel content")
	var kernel bytes.Buffer
	target := &registry.FilesTarget{Kernel: &kernel}
	desc := ocispec.Descriptor{
		MediaType:   registry.MimeTypeECIKernel,
		Size:        int64(len(payload)),
		Annotations: map[string]string{registry.AnnotationRole: registry.RoleKernel},
	}
	if err := target.Push(context.Background(), desc, bytes.NewReader(payload)); err == nil {
		t.Error("Push accepted a descriptor with no digest")
	}
}

func tarOf(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(content))}); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	return buf.Bytes()
}

func gzipOf(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(content); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}
