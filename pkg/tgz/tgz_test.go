package tgz_test

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/lf-edge/edge-containers/pkg/tgz"
)

func TestCompressUncompress(t *testing.T) {
	content := []byte("layer data")
	// Compress is given an absolute source path, as registry does.
	src := filepath.Join(t.TempDir(), "disk.qcow2")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"kernel", "kernel", false},
		{"initrd", "initrd", false},
		{"config", "config.json", false},
		{"root disk", "disk-root-disk.qcow2", false},
		{"nested", "a/b/disk.qcow2", false},
		{"absolute", "/etc/evil", true},
		{"absolute outside tempdir", filepath.Join(t.TempDir(), "evil"), true},
		{"dotdot", "../evil", true},
		{"inner dotdot", "a/../../evil", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := t.TempDir()
			outdir := filepath.Join(parent, "out")
			if err := os.Mkdir(outdir, 0o755); err != nil {
				t.Fatal(err)
			}
			tgzFile := filepath.Join(t.TempDir(), "layer.tgz")
			_, tgzSha, err := tgz.Compress(src, tt.entry, tgzFile, nil)
			if err != nil {
				t.Fatalf("Compress: %v", err)
			}
			written, err := os.ReadFile(tgzFile)
			if err != nil {
				t.Fatal(err)
			}
			if got := sha256.Sum256(written); string(got[:]) != string(tgzSha) {
				t.Fatalf("tgz hash %x does not match file hash %x", tgzSha, got)
			}

			err = tgz.Uncompress(tgzFile, outdir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Uncompress(%q) error = %v, wantErr %v", tt.entry, err, tt.wantErr)
			}
			if tt.wantErr {
				for _, escaped := range []string{filepath.Join(parent, "evil"), tt.entry} {
					if _, err := os.Stat(escaped); filepath.IsAbs(escaped) && err == nil {
						t.Fatalf("entry was written outside %s at %s", outdir, escaped)
					}
				}
				return
			}
			got, err := os.ReadFile(filepath.Join(outdir, tt.entry))
			if err != nil {
				t.Fatalf("reading extracted %s: %v", tt.entry, err)
			}
			if string(got) != string(content) {
				t.Fatalf("extracted %s = %q, want %q", tt.entry, got, content)
			}
		})
	}
}
