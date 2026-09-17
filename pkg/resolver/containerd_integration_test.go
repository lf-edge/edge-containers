package resolver_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/lf-edge/edge-containers/pkg/registry"
	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// containerdEnvVar names the address of a containerd to run these against, e.g.
// /home/user/tmp/ec-ctrd/c.sock. Skipped when unset, as the unit tests have no
// containerd to talk to.
const containerdEnvVar = "EC_TEST_CONTAINERD"

func containerdUnderTest(t *testing.T) string {
	t.Helper()
	addr := os.Getenv(containerdEnvVar)
	if addr == "" {
		t.Skipf("set %s to a containerd address to run this", containerdEnvVar)
	}
	return addr
}

// TestContainerdRoundTrip pushes an artifact into a containerd content store and
// pulls it back, which covers what an EVE device does on every local pull: the
// content store, the image service, and the GC labels that keep a manifest's
// children from being collected out from under it.
func TestContainerdRoundTrip(t *testing.T) {
	addr := containerdUnderTest(t)
	const namespace = "edge-containers-test"

	dir := t.TempDir()
	files := map[string][]byte{
		"kernel":   []byte("kernel bytes"),
		"initrd":   []byte("initrd bytes"),
		"root.raw": []byte("root disk bytes"),
	}
	paths := map[string]string{}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, content, 0644); err != nil {
			t.Fatalf("writing %s: %v", p, err)
		}
		paths[name] = p
	}
	ref := "example.com/eci/containerd-test:v1"

	ctx, res, err := ecresolver.NewContainerd(context.Background(), addr, namespace)
	if err != nil {
		t.Fatalf("connecting to containerd at %s: %v", addr, err)
	}
	defer func() { _ = res.Finalize(ctx) }()

	pusher := registry.Pusher{
		Artifact: &registry.Artifact{
			Kernel: &registry.FileSource{Path: paths["kernel"]},
			Initrd: &registry.FileSource{Path: paths["initrd"]},
			Root:   &registry.Disk{Source: &registry.FileSource{Path: paths["root.raw"]}, Type: registry.Raw},
		},
		Image: ref,
	}
	dig, err := pusher.Push(registry.FormatArtifacts, false, nil, registry.ConfigOpts{}, res)
	if err != nil {
		t.Fatalf("push: %v", err)
	}

	// containerd should now know the image by name, pointing at what we pushed
	client, err := containerd.New(addr)
	if err != nil {
		t.Fatalf("second containerd client: %v", err)
	}
	defer func() { _ = client.Close() }()
	nsCtx := namespaces.WithNamespace(context.Background(), namespace)
	image, err := client.ImageService().Get(nsCtx, ref)
	if err != nil {
		t.Fatalf("image service does not know %s: %v", ref, err)
	}
	if image.Target.Digest.String() != dig {
		t.Errorf("image points at %s, push reported %s", image.Target.Digest, dig)
	}
	if image.Target.MediaType != ocispec.MediaTypeImageManifest {
		t.Errorf("image media type = %s, want %s", image.Target.MediaType, ocispec.MediaTypeImageManifest)
	}

	// the manifest blob must carry a GC reference to each of its children, or
	// containerd is free to collect the layers this manifest still needs
	info, err := client.ContentStore().Info(nsCtx, image.Target.Digest)
	if err != nil {
		t.Fatalf("content info for the manifest: %v", err)
	}
	gcRefs := 0
	for k := range info.Labels {
		if len(k) > len("containerd.io/gc.ref.content") && k[:len("containerd.io/gc.ref.content")] == "containerd.io/gc.ref.content" {
			gcRefs++
		}
	}
	// three layers plus the config
	if gcRefs != 4 {
		t.Errorf("manifest carries %d gc refs, want 4 (labels: %v)", gcRefs, info.Labels)
	}

	// and pulling it back must reproduce the files
	ctx2, res2, err := ecresolver.NewContainerd(context.Background(), addr, namespace)
	if err != nil {
		t.Fatalf("second resolver: %v", err)
	}
	defer func() { _ = res2.Finalize(ctx2) }()

	var kernel, initrd, root bytes.Buffer
	puller := registry.Puller{Image: ref}
	desc, artifact, err := puller.Pull(&registry.FilesTarget{
		Kernel: &kernel, Initrd: &initrd, Root: &root,
	}, 0, false, nil, res2)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if desc.Digest.String() != dig {
		t.Errorf("pulled digest %s, pushed %s", desc.Digest, dig)
	}
	if artifact.Root == nil || artifact.Root.Type != registry.Raw {
		t.Errorf("pulled artifact root = %v, want a raw disk", artifact.Root)
	}
	for _, c := range []struct {
		name string
		got  *bytes.Buffer
		want []byte
	}{
		{"kernel", &kernel, files["kernel"]},
		{"initrd", &initrd, files["initrd"]},
		{"root", &root, files["root.raw"]},
	} {
		if !bytes.Equal(c.got.Bytes(), c.want) {
			t.Errorf("%s = %q, want %q", c.name, c.got.Bytes(), c.want)
		}
	}
	fmt.Fprintln(os.Stderr, "containerd round trip complete")
}
