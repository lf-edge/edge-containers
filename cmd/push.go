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
	formatStr  string
	disks      []string
	author     string
	osname     string
	arch       string
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
		var config registry.Source
		if configFile != "" {
			config = &registry.FileSource{Path: configFile}
		}
		artifact := &registry.Artifact{
			Kernel: &registry.FileSource{Path: kernelFile},
			Initrd: &registry.FileSource{Path: initrdFile},
			Root:   rootDisk,
			Config: config,
			Disks:  addlDisks,
		}
		pusher := registry.Pusher{
			Artifact: artifact,
			Image:    image,
		}
		// convert the format string into a proper format
		var format registry.Format
		switch formatStr {
		case "artifacts":
			format = registry.FormatArtifacts
		case "legacy":
			format = registry.FormatLegacy
		default:
			log.Fatalf("unknown format: %v", formatStr)
		}
		hash, err := pusher.Push(format, verbose, os.Stdout, registry.ConfigOpts{
			Author:       author,
			OS:           osname,
			Architecture: arch,
		}, remoteTarget)
		if err != nil {
			log.Fatalf("error pushing to registry: %v", err)
		}
		location := ""
		if remote != "" {
			location = fmt.Sprintf("to %s ", remote)
		}
		fmt.Printf("Pushed image %s %swith digest %s\n", image, location, hash)
	},
}

func pushInit() {
	pushCmd.Flags().StringVar(&kernelFile, "kernel", "", "path to kernel file, optional")
	pushCmd.Flags().StringVar(&initrdFile, "initrd", "", "path to initrd file, optional")
	pushCmd.Flags().StringVar(&rootFile, "root", "", "path to root disk file and type")
	pushCmd.Flags().StringVar(&configFile, "config", "", "path to ECI manifest config")
	pushCmd.Flags().StringVar(&author, "author", registry.DefaultAuthor, "author to use in generated config, if config not provided")
	pushCmd.Flags().StringVar(&osname, "OS", registry.DefaultOS, "os to use in generated config, if config not provided")
	pushCmd.Flags().StringVar(&arch, "arch", registry.DefaultArch, "arch to use in generated config, if config not provided")
	pushCmd.Flags().StringSliceVar(&disks, "disk", []string{}, "path to additional disk and type, may be invoked multiple times")
	pushCmd.Flags().StringVar(&formatStr, "format", "artifacts", "which format to use, one of: artifacts, legacy")
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
		Source: &registry.FileSource{Path: parts[0]},
		Type:   diskType,
	}, nil
}
