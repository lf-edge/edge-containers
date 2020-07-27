# Container Layout

When using the container format, the layout is as follows. Note that it can be anything,
with the annotations describing where the files are. However, when this utility
creates the container image, this is the layout it uses.

All files are in the root directory. They are:

```
/kernel
/initrd
/disk-root-<original_name>
/disk-<index>-<original_name>
```

These filenames match the filenames described [here](./filenames.md).

When these files are placed in the container image, annotations are added to the
manifest describing the paths to the various files. The annotations are described
in [annotations.md](./annotations.md).

We strongly recommend having each file be provided as a separate layer, to increase
reusability. A container image built via this utility always will be built that way.
However, we do not require it when reading an image.
