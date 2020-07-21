# ECI Distribution

This repository contains a golang library and CLI for ECI images, to  push to and pull from OCI registries.

It is inspired directly by [ORAS](https://github.com/deislabs/oras) and leverages it, but is opinionated to the ECI use case.

It uses elements of [OCI Artifacts](http://github.com/opencontainers/artifacts), but can use either standard mime-types and configs, or, where available, leverage full artifacts mime types.


## Usage

### Pushing an ECI

To push an ECI to a registry, you need the following items in a directory:

* a root disk image in any supported format: raw, vhd, vmdk, iso
* a Linux kernel (optional)
* a Linux initrd (optional)
* additional disks (optional)
* a config file, whose contents provide the desired OCI manifest config

You can push the image as follows:

```sh
eci push --root path/to/root.img:raw --kernel path/to/kernel --initrd path/to/initrd --disk path/to/disk1:iso --disk path/to/disk2:vmdk ... --config path/to/config lfedge/eci-nginx:ubuntu-1804-11715
```

The above assumes that the registry fully supports Artifacts and will use specialized mime types. If you wish to use an existing regstry that does
not yet support artifacts, pass the `--legacy` flag:

```sh
eci push --legacy --root path/to/root.img:raw --kernel path/to/kernel --initrd path/to/initrd --disk path/to/disk1:iso --disk path/to/disk2:vmdk ... --config path/to/config lfedge/eci-nginx:ubuntu-1804-11715
```

The `eci` command will take care of setting the correct mime types and annotations on all of the objects.

Note that disks, both root and additional, **must** have the file name, following by a `:` and the disk type, so that consumers know how to
interpret them, e.g. to send a disk file whose name is `mydisk` and is of type qcow2:

```sh
--disk mydisk:qcow2
```

### Pulling an ECI

To pull an ECI, you simply need a registry where the components will be downloaded:

```sh
eci pull lf-edge/eci-nginx:ubuntu-1804-11715
```

The above will default to placing artifacts in the current directory. To place them in a different directory:

```sh
eci pull --dir foo/bar/ lf-edge/eci-nginx:ubuntu-1804-11715
```

## Media Types and Annotations

The specific standard media types are at [docs/mediatypes.md](./docs/mediatypes.md).

In addition to the types, `eci` _always_ will add annotations to the layer and config in the manifest describing its purpose.

The specific standard annotations are at [docs/annotations.md](./docs/annotations.md).

## File Names

ECI is highly opinionated about the file names. No matter what names you pass to it, it will give the files particular names. These are listed in [docs/filenames.md](docs/filenames.md).

## Sample Manifest

A sample manifest for an actual pushed image is below. This is a manifest on docker hub, so the media types are the legacy types,
while the annotations provide the purpose.

```json
{
  "schemaVersion": 2,
  "config": {
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "sha256:ffb3941df4fe37f22165b124d66e966d93b3dbf2765b736818b57a4516aed94e",
    "size": 14,
    "annotations": {
      "org.lfedge.eci.mediaType": "application/vnd.lfedge.eci.config.v1+json",
      "org.opencontainers.image.title": "config.json"
    }
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:edeaaff3f1774ad2888673770c6d64097e391bc362d7d6fb34982ddf0efd18cb",
      "size": 4,
      "annotations": {
        "org.lfedge.eci.mediaType": "application/vnd.lfedge.eci.kernel.layer.v1+kernel",
        "org.lfedge.eci.role": "kernel",
        "org.opencontainers.image.title": "kernel"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:da1464fd7ceaf38ff56043bc1774af4fb5cb83ef5358981d78de0b8be5a6fbcb",
      "size": 4,
      "annotations": {
        "org.lfedge.eci.mediaType": "application/vnd.lfedge.eci.initrd.layer.v1+cpio",
        "org.lfedge.eci.role": "initrd",
        "org.opencontainers.image.title": "initrd"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:deb055d836e44a1dcf0317b0cacac2dbdd36301f82abf787f7849d3f5b916750",
      "size": 5,
      "annotations": {
        "org.lfedge.eci.mediaType": "application/vnd.lfedge.disk.layer.v1+raw",
        "org.lfedge.eci.role": "disk-root",
        "org.opencontainers.image.title": "disk-root-root.raw"
      }
    }
  ]
}
```

## Go Library

The go library is `github.com/lf-edge/edge-containers/pkg/registry`. Docs are available at [godoc.org/github.com/lf-edge/edge-containers/pkg/registry](https://godoc.org/github.com/lf-edge/edge-containers/pkg/registry).

## Build

The `eci` tool can be built via `make build`, which will deposit the build artifact in `dist/bin/eci-<os>-<arch>`, e.g. `dist/bin/eci-darwin-amd64` or `dist/bin/eci-linux-arm64`. To build it for alternate OSes or architectures, run:

```sh
make build OS=<target> ARCH=<target>
```

e.g.

```sh
make build OS=linux ARCH=amd64
make build OS=linux ARCH=amd64
```
