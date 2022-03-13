module github.com/lf-edge/edge-containers

go 1.15

require (
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/containerd/containerd v1.6.1
	github.com/coreos/etcd v3.3.27+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.14.4 // indirect
	github.com/moby/sys/mountinfo v0.6.0 // indirect
	github.com/moby/sys/signal v0.7.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	google.golang.org/genproto v0.0.0-20220302033224-9aa15565e42a // indirect
	oras.land/oras-go v1.1.0
	rsc.io/letsencrypt v0.0.3 // indirect
)

replace github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190826074503-38ab9da00309

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
