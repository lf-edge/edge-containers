# Annotations

Because many registries only support standard media types, we need a way to discover the purpose and format of a layer. The normal
way is to use the media type, but many such registries do not support it.

To resolve this issue, we use standard annotations in the manifest for each layer and config. In order to provide uniformity,
we provide these annotations even when the registry does support artifacts media types.

The annotations are as follows:

* `org.lfedge.eci.mediaType: <type>` - this will be identical to the [lfedge mediaType](./mediatypes.md) that is provided in the case of an artifacts-supporting registry
* `org.lfedge.eci.role: <role>` - for the role of this particular layer. Can be one of the following:
   * `kernel`
   * `initrd`
   * `disk-root`
   * `disk-additional` - for alternate non-root/boot disks
* `org.opencontainers.image.title: <name>` - the targeted name for the blob when stored on disk; see [filenames.md](./filenames.md)
