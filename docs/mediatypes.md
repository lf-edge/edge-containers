# Media Types

All artifacts in an OCI registry are required to have a media type. This is used both on the blob itself, as well as in the manifest that points to the
blobs.

When a registry supports it, i.e. the default artifacts mode, we use custom mediatypes to reflect the content. These are as follows:

* config: `application/vnd.lfedge.eci.config.v1+json`
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

When a registry does not support using custom mediatypes, we operate in legacy mode and use the following media types acceptable to all registries:

* config: `application/vnd.oci.image.config.v1+json`
* layers: `application/vnd.oci.image.layer.v1.tar`

