package tgz

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// Compress takes a given path to a file and creates a tgz file that
// contains only that file. Gives the file the provided name in the tgz.
// Returns hashes of the tar and the entire gzip
func Compress(infile, name, outfile string) (tarSha []byte, tgzSha []byte, err error) {
	tgzHasher, tarHasher := sha256.New(), sha256.New()
	tgzfile, err := os.Create(outfile)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not create tgz file '%s': %v", outfile, err)
	}
	defer tgzfile.Close()
	gzipWriter := gzip.NewWriter(io.MultiWriter(tgzfile, tgzHasher))
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(io.MultiWriter(gzipWriter, tarHasher))
	defer tarWriter.Close()
	if err := addFileToTarWriter(infile, name, tarWriter); err != nil {
		return nil, nil, fmt.Errorf("could not add %s to tar as %s: %v", infile, name, err)
	}
	// we cannot wait for the defer, since we have to Close() to flush
	// everything out before calculating final hashes in the return line
	tarWriter.Close()
	gzipWriter.Close()
	return tarHasher.Sum(nil), tgzHasher.Sum(nil), nil
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
