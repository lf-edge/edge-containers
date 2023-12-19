module github.com/lf-edge/edge-containers

go 1.15

require (
	github.com/containerd/containerd v1.6.26
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/moby/sys/signal v0.7.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.8.4
	oras.land/oras-go v1.1.0
)

replace github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190826074503-38ab9da00309

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
