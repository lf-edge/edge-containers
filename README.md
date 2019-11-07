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

In legacy mode, the `config.mediaType` is `application/vnd.oci.image.config.v1+json` while all layers are `application/vnd.oci.image.layer.v1.tar`. This are the only acceptable types for registries that do not yet support artifactas. 

In artifacts mode (the default), the media types are as follows:

* `config`: `application/vnd.lfedge.eci.config.v1+json `
* kernel: `application/vnd.lfedge.eci.kernel.layer.v1.tar`
* initrd: `application/vnd.lfedge.eci.initrd.layer.v1.tar`
* disks: disks always have a media type that conforms to their format
  * raw: `application/vnd.lfedge.disk.layer.v1+raw`
  * vhd: `application/vnd.lfedge.disk.layer.v1+vhd`
  * vmdk: `application/vnd.lfedge.disk.layer.v1+vmdk`
  * iso: `application/vnd.lfedge.disk.layer.v1+iso`
  * qcow: `application/vnd.lfedge.disk.layer.v1+qcow`
  * qcow2: `application/vnd.lfedge.disk.layer.v1+qcow2`
  * ova: `application/vnd.lfedge.disk.layer.v1+ova`
  * vhdx: `application/vnd.lfedge.disk.layer.v1+vhdx`

In addition to the types, when available via an artifacts-supporting registry, `eci` _always_ will add annotations to the layer describing its purpose.
The annotations are as follows:

* `org.lfedge.eci.mediaType: <type>` - this will be identical to the mediaType that is provided in the case of an artifacts-supporting registry
* `org.lfedge.eci.role: <role>` - for the role of this particular layer. Can be one of the following:
   * `kernel`
   * `initrd`
   * `disk-root`
   * `disk-additional` - for alternate non-root/boot disks

