package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lf-edge/edge-containers/pkg/registry"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	kernelFile string
	initrdFile string
	rootFile   string
	configFile string
	legacy     bool
	disks      []string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push an ECI to a registry",
	Long:  `push an Edge Container Image (ECI) to an OCI compliant registry`,
	Run: func(cmd *cobra.Command, args []string) {
		// must be exactly one arg, the URL to the manifest
		if len(args) != 1 {
			log.Fatal("must be exactly one arg, run help")
		}
		image := args[0]
		// convert the disks to Disk struct
		var (
			rootDisk *registry.Disk
			err      error
		)
		if rootFile != "" {
			rootDisk, err = diskToStruct(rootFile)
			if err != nil {
				log.Fatalf("unable to read root disk %s: %v", rootFile, err)
			}
		}
		addlDisks := make([]*registry.Disk, 0, len(disks))
		for _, d := range disks {
			disk, err := diskToStruct(d)
			if err != nil {
				log.Fatalf("unable to read disk %s: %v", d, err)
			}
			addlDisks = append(addlDisks, disk)
		}
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		// construct and pass along
		artifact := &registry.Artifact{
			Kernel: kernelFile,
			Initrd: initrdFile,
			Root:   rootDisk,
			Config: configFile,
			Legacy: legacy,
			Disks:  addlDisks,
		}
		pusher := registry.Pusher{
			Artifact: artifact,
			Image: image,
		}
		hash, err := pusher.Push(verbose, os.Stdout)
		if err != nil {
			log.Fatalf("error pushing to registry: %v", err)
		}
		fmt.Printf("Pushed image %s with digest %s\n", image, hash)

	},
}

func pushInit() {
	pushCmd.Flags().StringVar(&kernelFile, "kernel", "", "path to kernel file, optional")
	pushCmd.Flags().StringVar(&initrdFile, "initrd", "", "path to initrd file, optional")
	pushCmd.Flags().StringVar(&rootFile, "root", "", "path to root disk file and type")
	pushCmd.Flags().StringVar(&configFile, "config", "", "path to ECI manifest config")
	pushCmd.Flags().StringSliceVar(&disks, "disk", []string{}, "path to additional disk and type, may be invoked multiple times")
	pushCmd.Flags().BoolVar(&legacy, "legacy", false, "whether to work in legacy mode with registries that do not support OCI Artifacts")
	pushCmd.Flags().BoolVar(&debug, "debug", false, "debug output")
	pushCmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
}

// convert a "path:type" to a Disk struct
func diskToStruct(path string) (*registry.Disk, error) {
	parts := strings.SplitN(path, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected structure <path>:<type>")
	}
	// get the disk type
	diskType, ok := registry.NameToType[parts[1]]
	if !ok {
		return nil, fmt.Errorf("unknown disk type: %s", parts[1])
	}
	return &registry.Disk{
		Path: parts[0],
		Type: diskType,
	}, nil
}
