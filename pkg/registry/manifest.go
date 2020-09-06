package registry

import (
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/lf-edge/edge-containers/pkg/store"
	"github.com/lf-edge/edge-containers/pkg/tgz"

	"github.com/deislabs/oras/pkg/content"

	ctrcontent "github.com/containerd/containerd/content"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Manifest create the manifest for the given Artifact.
func (a Artifact) Manifest(format Format, configOpts ConfigOpts, legacyOpts ...LegacyOpt) (*ocispec.Manifest, ctrcontent.Provider, error) {
	var (
		desc            ocispec.Descriptor
		mediaType       string
		customMediaType string
		role            string
		name            string
		filepath        string
		err             error
		lOpts           = legacyInfo{}
	)

	for _, o := range legacyOpts {
		o(&lOpts)
	}

	// Go through each file type in the registry and add the appropriate file type and path, along with annotations
	fileStore := content.NewFileStore("")
	defer fileStore.Close()
	memStore := content.NewMemoryStore()
	multiStore := store.MultiReader{}
	multiStore.AddStore(fileStore, memStore)

	// if we have the container format, we need to create tgz layers
	var (
		tmpDir       string
		labels       = map[string]string{}
		pushContents = []ocispec.Descriptor{}
		layers       = []digest.Digest{}
		layerHash    digest.Digest
	)

	if format == FormatLegacy {
		tmpDir = lOpts.tmpdir
		if tmpDir == "" {
			return nil, nil, fmt.Errorf("did not provide valid temporary directory for format legacy")
		}
	}

	if a.Kernel != "" {
		role = RoleKernel
		name = "kernel"
		layerHash = ""
		customMediaType = MimeTypeECIKernel
		filepath = a.Kernel
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatLegacy {
			tgzfile := path.Join(tmpDir, name)
			tarHash, _, err := tgz.Compress(filepath, name, tgzfile, lOpts.timestamp)
			if err != nil {
				return nil, nil, fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
			// convert the tarHash into a digest
			layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return nil, nil, fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
		if layerHash == "" {
			layerHash = desc.Digest
		}
		layers = append(layers, layerHash)

		labels[AnnotationKernelPath] = fmt.Sprintf("/%s", name)
	}

	if a.Initrd != "" {
		role = RoleInitrd
		name = "initrd"
		layerHash = ""
		customMediaType = MimeTypeECIInitrd
		filepath = a.Initrd
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatLegacy {
			tgzfile := path.Join(tmpDir, name)
			tarHash, _, err := tgz.Compress(filepath, name, tgzfile, lOpts.timestamp)
			if err != nil {
				return nil, nil, fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
			layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return nil, nil, fmt.Errorf("error adding %s at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
		if layerHash == "" {
			layerHash = desc.Digest
		}
		layers = append(layers, layerHash)

		labels[AnnotationInitrdPath] = fmt.Sprintf("/%s", name)
	}

	if disk := a.Root; disk != nil {
		role = RoleRootDisk
		customMediaType = TypeToMime[disk.Type]
		filepath = disk.Path
		name := fmt.Sprintf("disk-root-%s", path.Base(filepath))
		layerHash = ""
		mediaType = GetLayerMediaType(customMediaType, format)
		if format == FormatLegacy {
			tgzfile := path.Join(tmpDir, name)
			tarHash, _, err := tgz.Compress(filepath, name, tgzfile, lOpts.timestamp)
			if err != nil {
				return nil, nil, fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
			}
			filepath = tgzfile
			layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
		}
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return nil, nil, fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
		desc.Annotations[AnnotationRole] = role
		desc.Annotations[ocispec.AnnotationTitle] = name
		pushContents = append(pushContents, desc)
		if layerHash == "" {
			layerHash = desc.Digest
		}
		layers = append(layers, layerHash)

		labels[AnnotationRootPath] = fmt.Sprintf("/%s", name)
	}
	for i, disk := range a.Disks {
		if disk != nil {
			role = RoleAdditionalDisk
			customMediaType = TypeToMime[disk.Type]
			filepath = disk.Path
			name := fmt.Sprintf("disk-%d-%s", i, path.Base(filepath))
			layerHash = ""
			mediaType = GetLayerMediaType(customMediaType, format)
			if format == FormatLegacy {
				tgzfile := path.Join(tmpDir, name)
				tarHash, _, err := tgz.Compress(filepath, name, tgzfile, lOpts.timestamp)
				if err != nil {
					return nil, nil, fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
				}
				filepath = tgzfile
				layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
			}
			desc, err = fileStore.Add(name, mediaType, filepath)
			if err != nil {
				return nil, nil, fmt.Errorf("error adding %s disk at %s: %v", name, filepath, err)
			}
			desc.Annotations[AnnotationMediaType] = customMediaType
			desc.Annotations[AnnotationRole] = role
			desc.Annotations[ocispec.AnnotationTitle] = name
			pushContents = append(pushContents, desc)
			if layerHash == "" {
				layerHash = desc.Digest
			}
			layers = append(layers, layerHash)

			labels[fmt.Sprintf(AnnotationDiskIndexPathPattern, i)] = fmt.Sprintf("/%s", name)
		}
	}
	for _, other := range a.Other {
		if other != "" {
			customMediaType = MimeTypeECIOther
			filepath = other
			name = path.Base(filepath)
			layerHash = ""
			mediaType = GetLayerMediaType(customMediaType, format)
			if format == FormatLegacy {
				tgzfile := path.Join(tmpDir, name)
				tarHash, _, err := tgz.Compress(filepath, name, tgzfile, lOpts.timestamp)
				if err != nil {
					return nil, nil, fmt.Errorf("error creating tgz file for %s: %v", filepath, err)
				}
				filepath = tgzfile
				layerHash = digest.NewDigestFromBytes(digest.SHA256, tarHash)
			}
			desc, err = fileStore.Add(name, mediaType, filepath)
			if err != nil {
				return nil, nil, fmt.Errorf("error adding %s other at %s: %v", name, filepath, err)
			}
			desc.Annotations[AnnotationMediaType] = customMediaType
			pushContents = append(pushContents, desc)
			if layerHash == "" {
				layerHash = desc.Digest
			}
			layers = append(layers, layerHash)

			labels[AnnotationOther] = fmt.Sprintf("/%s", name)
		}
	}

	// was a config specified?
	if a.Config != "" {
		name = "config.json"
		customMediaType = MimeTypeECIConfig
		filepath = a.Config
		mediaType = GetConfigMediaType(customMediaType, format)
		desc, err = fileStore.Add(name, mediaType, filepath)
		if err != nil {
			return nil, nil, fmt.Errorf("error adding %s config at %s: %v", name, filepath, err)
		}
		desc.Annotations[AnnotationMediaType] = customMediaType
	} else {
		// for container format, we expect to have a specific config so docker can work with it
		created := time.Now()
		configAuthor, configOS, configArch := configOpts.Author, configOpts.OS, configOpts.Architecture
		if configAuthor == "" {
			configAuthor = DefaultAuthor
		}
		if configOS == "" {
			configOS = DefaultOS
		}
		if configArch == "" {
			configArch = DefaultArch
		}
		config := ocispec.Image{
			Created:      &created,
			Author:       configAuthor,
			Architecture: configArch,
			OS:           configOS,
			RootFS: ocispec.RootFS{
				Type:    "layers",
				DiffIDs: layers,
			},
			Config: ocispec.ImageConfig{
				Labels: labels,
			},
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			return nil, nil, fmt.Errorf("error marshaling config to json: %v", err)
		}

		name = "config.json"
		mediaType = MimeTypeOCIImageConfig
		desc = memStore.Add(name, mediaType, configBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("error adding OCI config: %v", err)
		}
	}
	// make our manifest
	return &ocispec.Manifest{
		Config: desc,
		Layers: pushContents,
	}, multiStore, nil
}
