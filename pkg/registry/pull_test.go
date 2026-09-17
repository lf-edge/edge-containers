package registry_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/lf-edge/edge-containers/pkg/registry"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/content/oci"
)

// TestPushPullRoundTrip pushes an artifact into an in-memory store and pulls it
// back out into writers, which covers the whole path: building the manifest,
// streaming blobs out, and routing each one to the writer that wants it. The two
// formats take different routes on the way back -- artifacts is a blob per file,
// legacy a gzipped tar split by the paths the config names.
func TestPushPullRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format registry.Format
	}{
		{"artifacts", registry.FormatArtifacts},
		{"legacy", registry.FormatLegacy},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			inputs := map[string]TestInputFile{
				"kernel": NewTestInputFile("kernel", "kernel", tmpdir),
				"initrd": NewTestInputFile("initrd", "initrd", tmpdir),
				"root":   NewTestInputFile("root.raw", "disk-root-root.raw", tmpdir),
				"disk1":  NewTestInputFile("disk1.qcow2", "disk-0-disk1.qcow2", tmpdir),
			}
			for _, v := range inputs {
				if err := os.WriteFile(v.Fullname(), v.Contents(), 0644); err != nil {
					t.Fatalf("unable to create %s: %v", v.Fullname(), err)
				}
			}

			store := memory.New()
			_, resolver, err := ecresolver.NewResolver(context.Background(), store)
			if err != nil {
				t.Fatalf("unexpected error creating resolver: %v", err)
			}

			pusher := registry.Pusher{
				Artifact: &registry.Artifact{
					Kernel: &registry.FileSource{Path: inputs["kernel"].Fullname()},
					Initrd: &registry.FileSource{Path: inputs["initrd"].Fullname()},
					Root:   &registry.Disk{Source: &registry.FileSource{Path: inputs["root"].Fullname()}, Type: rootDiskType},
					Disks:  []*registry.Disk{{Source: &registry.FileSource{Path: inputs["disk1"].Fullname()}, Type: diskOneType}},
				},
				Image:     testImageName,
				Timestamp: &initTime,
			}
			if _, err := pusher.Push(tt.format, false, nil, registry.ConfigOpts{}, resolver); err != nil {
				t.Fatalf("push: %v", err)
			}

			var kernel, initrd, root, disk1 bytes.Buffer
			target := &registry.FilesTarget{
				Kernel: &kernel,
				Initrd: &initrd,
				Root:   &root,
				Disks:  []io.Writer{&disk1},
			}

			puller := registry.Puller{Image: testImageName}
			desc, artifact, err := puller.Pull(target, 0, false, nil, resolver)
			if err != nil {
				t.Fatalf("pull: %v", err)
			}
			if desc == nil || desc.Digest == "" {
				t.Fatal("pull returned no descriptor")
			}

			for _, c := range []struct {
				name string
				got  *bytes.Buffer
				want []byte
			}{
				{"kernel", &kernel, inputs["kernel"].Contents()},
				{"initrd", &initrd, inputs["initrd"].Contents()},
				{"root", &root, inputs["root"].Contents()},
			} {
				if !bytes.Equal(c.got.Bytes(), c.want) {
					t.Errorf("%s = %q, want %q", c.name, c.got.Bytes(), c.want)
				}
			}

			if artifact.Kernel == nil || artifact.Kernel.GetPath() != "kernel" {
				t.Errorf("artifact kernel = %v, want kernel", artifact.Kernel)
			}
			if artifact.Initrd == nil || artifact.Initrd.GetPath() != "initrd" {
				t.Errorf("artifact initrd = %v, want initrd", artifact.Initrd)
			}
			if artifact.Root == nil {
				t.Fatal("artifact has no root disk")
			}
			if artifact.Root.Type != rootDiskType {
				t.Errorf("artifact root type = %v, want %v", artifact.Root.Type, rootDiskType)
			}
			if len(artifact.Disks) != 1 {
				t.Fatalf("artifact has %d additional disks, want 1", len(artifact.Disks))
			}
			if artifact.Disks[0].Type != diskOneType {
				t.Errorf("artifact disk type = %v, want %v", artifact.Disks[0].Type, diskOneType)
			}

			// An additional disk reaches its writer only in the legacy format, where
			// the config names its path inside the layer. The artifacts format routes
			// by role, and no role writer is wired up for additional disks.
			wantDisk := []byte(nil)
			if tt.format == registry.FormatLegacy {
				wantDisk = inputs["disk1"].Contents()
			}
			if !bytes.Equal(disk1.Bytes(), wantDisk) {
				t.Errorf("additional disk = %q, want %q", disk1.Bytes(), wantDisk)
			}
		})
	}
}

// TestPullRejectsEmptyImage checks the argument guard before anything is contacted.
func TestPullRejectsEmptyImage(t *testing.T) {
	store := memory.New()
	_, resolver, err := ecresolver.NewResolver(context.Background(), store)
	if err != nil {
		t.Fatalf("unexpected error creating resolver: %v", err)
	}
	puller := registry.Puller{}
	_, _, err = puller.Pull(&registry.FilesTarget{}, 0, false, nil, resolver)
	if err == nil || !strings.Contains(err.Error(), "must have valid image ref") {
		t.Errorf("error = %v, want it to mention a valid image ref", err)
	}
}

// TestPullTwiceIntoSameStore pulls unchanged content into a destination that
// already holds it. A destination that keeps what it is given -- an OCI layout on
// disk, which is what resolver.Directory hands over -- already has every blob the
// second pull offers, and that is a no-op rather than a failure.
func TestPullTwiceIntoSameStore(t *testing.T) {
	tmpdir := t.TempDir()
	kernel := NewTestInputFile("kernel", "kernel", tmpdir)
	if err := os.WriteFile(kernel.Fullname(), kernel.Contents(), 0644); err != nil {
		t.Fatalf("unable to create %s: %v", kernel.Fullname(), err)
	}

	store := memory.New()
	_, resolver, err := ecresolver.NewResolver(context.Background(), store)
	if err != nil {
		t.Fatalf("unexpected error creating resolver: %v", err)
	}
	pusher := registry.Pusher{
		Artifact:  &registry.Artifact{Kernel: &registry.FileSource{Path: kernel.Fullname()}},
		Image:     testImageName,
		Timestamp: &initTime,
	}
	if _, err := pusher.Push(registry.FormatArtifacts, false, nil, registry.ConfigOpts{}, resolver); err != nil {
		t.Fatalf("push: %v", err)
	}

	dst, err := oci.New(t.TempDir())
	if err != nil {
		t.Fatalf("unable to create oci store: %v", err)
	}
	puller := registry.Puller{Image: testImageName}
	for i := 1; i <= 2; i++ {
		if _, _, err := puller.Pull(dst, 0, false, nil, resolver); err != nil {
			t.Fatalf("pull %d: %v", i, err)
		}
	}
}
