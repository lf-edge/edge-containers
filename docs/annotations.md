# Annotations

Because many registries only support standard media types, we need a way to discover the purpose and format of a layer. The normal
way is to use the media type, but many such registries do not support it.

To resolve this issue, we use standard annotations in the manifest for each layer and config. In order to
provide uniformity, we provide these annotations even when the registry does support artifacts media types.

The annotations are as follows:

* `org.lfedge.eci.mediaType: <type>` - this will be identical to the [lfedge mediaType](./mediatypes.md) that is provided in the case of an artifacts-supporting registry
* `org.lfedge.eci.role: <role>` - for the role of this particular layer. Can be one of the following:
   * `kernel`
   * `initrd`
   * `disk-root`
   * `disk-additional` - for alternate non-root/boot disks
* `org.opencontainers.image.title: <name>` - the targeted name for the blob when stored on disk; see [filenames.md](./filenames.md)

In addition, there are [manifest annotations](https://github.com/opencontainers/image-spec/blob/master/manifest.md)
to describe the paths to files when using the container image format. These annotations are:

* `org.lfedge.eci.artifact.kernel: <path>` - path to the kernel file
* `org.lfedge.eci.artifact.initrd: <path>` - path to the initrd file
* `org.lfedge.eci.artifact.root: <path>` - path to the root disk
* `org.lfedge.eci.artifact.disk-<index>: <path>` - path to the an additional indexed disk
