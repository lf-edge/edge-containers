package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/lf-edge/edge-containers/pkg/registry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/pkg/content"
)

var (
	kernel   string
	initrd   string
	config   string
	rootDisk string
)

var pullFilesCmd = &cobra.Command{
	Use:   "pullfiles",
	Short: "pull an ECI from a registry, placing each artifact into a different target file location",
	Long:  `pull an Edge Container Image (ECI) from an OCI compliant registry, extracting each artifact individually`,
	Run: func(cmd *cobra.Command, args []string) {
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		// must be exactly one arg, the URL to the manifest
		if len(args) != 1 {
			log.Fatal("must be exactly one arg, the name of the image to download")
		}
		image := args[0]
		puller := registry.Puller{
			Image: image,
		}
		target := &registry.FilesTarget{}
		if kernel != "" {
			f, err := os.OpenFile(kernel,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open kernel file %s for writing: %v", kernel, err)
			}
			defer f.Close()
			target.Kernel = f
		}
		if config != "" {
			f, err := os.OpenFile(config,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open config file %s for writing: %v", config, err)
			}
			defer f.Close()
			target.Config = f
		}
		if initrd != "" {
			f, err := os.OpenFile(initrd,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open initrd file %s for writing: %v", initrd, err)
			}
			defer f.Close()
			target.Initrd = f
		}
		if rootDisk != "" {
			f, err := os.OpenFile(rootDisk,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open root disk file %s for writing: %v", rootDisk, err)
			}
			defer f.Close()
			target.Root = f
		}
		desc, artifact, err := puller.Pull(target, blocksize, verbose, os.Stdout, remoteTarget)
		if err != nil {
			log.Fatalf("error pulling from registry: %v", err)
		}
		fmt.Printf("Pulled image %s with digest %s\n", image, string(desc.Digest))
		fmt.Println("file locations and types:")
		if kernel != "" {
			fmt.Printf("\tkernel: %v\n", artifact.Kernel.GetPath())
		}
		if initrd != "" {
			fmt.Printf("\tinitrd: %s\n", artifact.Initrd.GetPath())
		}
		if rootDisk != "" {
			root := artifact.Root
			if root == nil {
				fmt.Printf("\troot: \n")
			} else {
				fmt.Printf("\troot: %s %v\n", root.Source.GetPath(), root.Type)
			}
		}
		for i, d := range artifact.Disks {
			fmt.Printf("\tadditional disk %d: %s %v\n", i, d.Source.GetPath(), d.Type)
		}
	},
}

func pullFilesInit() {
	pullFilesCmd.Flags().StringVar(&kernel, "kernel", "", "path to place kernel")
	pullFilesCmd.Flags().StringVar(&config, "config", "", "path to place image config")
	pullFilesCmd.Flags().StringVar(&initrd, "initrd", "", "path to place initrd")
	pullFilesCmd.Flags().StringVar(&rootDisk, "root", "", "path to place root disk")
	pullFilesCmd.Flags().IntVar(&blocksize, "blocksize", content.DefaultBlocksize, "blocksize to use for gunzip/untar")
	pullFilesCmd.Flags().BoolVar(&debug, "debug", false, "debug output")
	pullFilesCmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
}
