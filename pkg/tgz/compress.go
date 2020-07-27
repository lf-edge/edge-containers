package tgz

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// Compress takes a given path to a file and creates a tgz file that
// contains only that file. Gives the file the provided name in the tgz.
func Compress(infile, name, outfile string) error {
	tgzfile, err := os.Create(outfile)
	if err != nil {
		return fmt.Errorf("Could not create tgz file '%s': %v", outfile, err)
	}
	defer tgzfile.Close()
	gzipWriter := gzip.NewWriter(tgzfile)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	if err := addFileToTarWriter(infile, name, tarWriter); err != nil {
		return fmt.Errorf("could not add %s to tar as %s: %v", infile, name, err)
	}
	return nil
}

func addFileToTarWriter(infile, name string, tarWriter *tar.Writer) error {
	file, err := os.Open(infile)
	if err != nil {
		return fmt.Errorf("could not open %s for reading: %v", infile, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Could not stat '%s': %v", infile, err)
	}

	// create the header
	header := &tar.Header{
		Name:    name,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header for '%s': %v", infile, err)
	}
	if _, err = io.Copy(tarWriter, file); err != nil {
		return fmt.Errorf("error writing '%s' data to tar: %v", infile, err)
	}
	return nil
}
