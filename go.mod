module github.com/lf-edge/edge-containers

go 1.15

require (
	github.com/containerd/containerd v1.6.12
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.14.4 // indirect
	github.com/moby/sys/mountinfo v0.6.0 // indirect
	github.com/moby/sys/signal v0.7.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	oras.land/oras-go v1.1.0
)

replace github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190826074503-38ab9da00309

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
