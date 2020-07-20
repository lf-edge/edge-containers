# Filenames

The various artifacts, i.e. arbitrary blobs, can be stored locally as files, normally
performed via `eci pull`. These may be required to have specific file names. The targeted
filenames are stored in the [annotations](./annotations.md) using the official
[ocispec.AnnotationTitle](https://godoc.org/github.com/opencontainers/image-spec/specs-go/v1#pkg-constants)
annotation.

The standard filenames used are:

* kernel: `kernel`
* initrd: `initrd`
* root disk: `disk-root-<original_name>`, e.g. if the file was `rootdisk.iso`, then the file will be `disk-root-rootdisk.iso`
* additional disks: `disk-<index>-<original_name>`, e.g. if the original file was `foo.qcow2`, then the file will be `disk-0-foo.qcow2`

The purpose of the disk naming is to preserve the filename extensions, which may matter to a consumer,
while enforcing a standard for disk order and discovery.
