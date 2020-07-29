module github.com/lf-edge/edge-containers

go 1.12

require (
	github.com/containerd/containerd v1.3.0
	github.com/containerd/fifo v0.0.0-20200410184934-f15a3290365b // indirect
	github.com/containerd/ttrpc v1.0.1 // indirect
	github.com/containerd/typeurl v1.0.1 // indirect
	github.com/deislabs/oras v0.7.0
	github.com/docker/docker v0.7.3-0.20190826074503-38ab9da00309
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
)

replace github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190826074503-38ab9da00309
