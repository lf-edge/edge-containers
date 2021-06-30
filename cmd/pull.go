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
	pullDir string
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "pull an ECI from a registry to a local directory",
	Long:  `pull an Edge Container Image (ECI) from an OCI compliant registry`,
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
		desc, artifact, err := puller.Pull(registry.DirTarget{Dir: pullDir}, blocksize, verbose, os.Stdout, remoteTarget)
		if err != nil {
			log.Fatalf("error pulling from registry: %v", err)
		}
		fmt.Printf("Pulled image %s with digest %s to directory %s\n", image, string(desc.Digest), pullDir)
		fmt.Println("file locations and types:")
		fmt.Printf("\tkernel: %s\n", artifact.Kernel)
		fmt.Printf("\tinitrd: %s\n", artifact.Initrd)
		rootDisk := artifact.Root
		if rootDisk == nil {
			fmt.Printf("\troot: \n")
		} else {
			fmt.Printf("\troot: %s %v\n", rootDisk.Source.GetPath(), rootDisk.Type)
		}
		for i, d := range artifact.Disks {
			fmt.Printf("\tadditional disk %d: %s %v\n", i, d.Source.GetPath(), d.Type)
		}
	},
}

func pullInit() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	pullCmd.Flags().StringVar(&pullDir, "dir", cwd, "directory where to install the ECI, optional")
	pullCmd.Flags().IntVar(&blocksize, "blocksize", content.DefaultBlocksize, "blocksize to use for gunzip/untar")
	pullCmd.Flags().BoolVar(&debug, "debug", false, "debug output")
	pullCmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
}
