package registry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/lf-edge/edge-containers/pkg/registry"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// registryUnderTest is the host:port of a plain-HTTP OCI registry to run the
// integration tests against, e.g. 127.0.0.1:5199. These tests are skipped when it
// is unset, as they need a registry the unit tests do not.
const registryEnvVar = "EC_TEST_REGISTRY"

func registryUnderTest(t *testing.T) string {
	t.Helper()
	addr := os.Getenv(registryEnvVar)
	if addr == "" {
		t.Skipf("set %s to a plain-HTTP registry host:port to run this", registryEnvVar)
	}
	return addr
}

// TestRegistryRoundTrip pushes an artifact to a real registry and pulls it back,
// which is the one path the in-memory tests cannot cover: the HTTP transport, the
// credential client and the manifest and blob endpoints.
func TestRegistryRoundTrip(t *testing.T) {
	addr := registryUnderTest(t)
	for _, tt := range []struct {
		name   string
		format registry.Format
		repo   string
	}{
		{"artifacts", registry.FormatArtifacts, "eci-artifacts"},
		{"legacy", registry.FormatLegacy, "eci-legacy"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			inputs := map[string]TestInputFile{
				"kernel": NewTestInputFile("kernel", "kernel", tmpdir),
				"initrd": NewTestInputFile("initrd", "initrd", tmpdir),
				"root":   NewTestInputFile("root.raw", "disk-root-root.raw", tmpdir),
			}
			for _, v := range inputs {
				if err := os.WriteFile(v.Fullname(), v.Contents(), 0644); err != nil {
					t.Fatalf("unable to create %s: %v", v.Fullname(), err)
				}
			}
			ref := fmt.Sprintf("%s/%s:v1", addr, tt.repo)

			ctx := context.Background()
			_, res, err := ecresolver.NewRegistryWithOpts(ctx, ecresolver.WithPlainHTTP())
			if err != nil {
				t.Fatalf("creating resolver: %v", err)
			}
			pusher := registry.Pusher{
				Artifact: &registry.Artifact{
					Kernel: &registry.FileSource{Path: inputs["kernel"].Fullname()},
					Initrd: &registry.FileSource{Path: inputs["initrd"].Fullname()},
					Root:   &registry.Disk{Source: &registry.FileSource{Path: inputs["root"].Fullname()}, Type: rootDiskType},
				},
				Image:     ref,
				Timestamp: &initTime,
			}
			dig, err := pusher.Push(tt.format, false, nil, registry.ConfigOpts{}, res)
			if err != nil {
				t.Fatalf("push: %v", err)
			}

			// the registry itself should now serve that manifest by tag
			manifestURL := fmt.Sprintf("http://%s/v2/%s/manifests/v1", addr, tt.repo)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
			if err != nil {
				t.Fatalf("building request: %v", err)
			}
			req.Header.Set("Accept", ocispec.MediaTypeImageManifest)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("fetching manifest: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("manifest fetch returned %d", resp.StatusCode)
			}
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("reading manifest: %v", err)
			}
			var manifest ocispec.Manifest
			if err := json.Unmarshal(raw, &manifest); err != nil {
				t.Fatalf("unmarshalling manifest: %v", err)
			}
			if got := resp.Header.Get("Docker-Content-Digest"); got != dig {
				t.Errorf("registry reports digest %s, push reported %s", got, dig)
			}
			if len(manifest.Layers) != 3 {
				t.Fatalf("manifest has %d layers, want 3", len(manifest.Layers))
			}

			// and pulling it back must reproduce the files byte for byte
			var kernel, initrd, root bytes.Buffer
			puller := registry.Puller{Image: ref}
			if _, artifact, err := puller.Pull(&registry.FilesTarget{
				Kernel: &kernel, Initrd: &initrd, Root: &root,
			}, 0, false, nil, res); err != nil {
				t.Fatalf("pull: %v", err)
			} else if artifact.Root == nil || artifact.Root.Type != rootDiskType {
				t.Errorf("pulled artifact root = %v, want type %v", artifact.Root, rootDiskType)
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
		})
	}
}

// TestRegistryRequiresPlainHTTP is the negative control for WithPlainHTTP: without
// it the resolver speaks HTTPS, which a registry served without TLS refuses.
func TestRegistryRequiresPlainHTTP(t *testing.T) {
	addr := registryUnderTest(t)
	tmpdir := t.TempDir()
	kernel := NewTestInputFile("kernel", "kernel", tmpdir)
	if err := os.WriteFile(kernel.Fullname(), kernel.Contents(), 0644); err != nil {
		t.Fatalf("unable to create %s: %v", kernel.Fullname(), err)
	}

	ctx := context.Background()
	_, res, err := ecresolver.NewRegistry(ctx)
	if err != nil {
		t.Fatalf("creating resolver: %v", err)
	}
	pusher := registry.Pusher{
		Artifact:  &registry.Artifact{Kernel: &registry.FileSource{Path: kernel.Fullname()}},
		Image:     fmt.Sprintf("%s/eci-https:v1", addr),
		Timestamp: &initTime,
	}
	if _, err := pusher.Push(registry.FormatArtifacts, false, nil, registry.ConfigOpts{}, res); err == nil {
		t.Error("push over HTTPS to a plain-HTTP registry succeeded, so WithPlainHTTP is not doing anything")
	}
}
