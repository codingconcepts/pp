package fs

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// MakeExecutable runs +x against a given file path.
func MakeExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	// Add +x permissions.
	mode := info.Mode()
	executableMode := mode | 0111

	if err := os.Chmod(path, executableMode); err != nil {
		return fmt.Errorf("applying file permissions: %w", err)
	}

	return nil
}

// ExtractBinary extracts a file from a given tarPath and writes it to
// a file called destName.
func ExtractBinary(tarPath, destName string) (err error) {
	defer func() {
		if cerr := os.Remove(tarPath); err != nil {
			err = cerr
		}
	}()

	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("opening tar file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	if _, err = tarReader.Next(); err == io.EOF {
		return fmt.Errorf("archive is empty")
	}
	if err != nil {
		return fmt.Errorf("reading tar file header: %w", err)
	}

	out, err := os.OpenFile(destName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, tarReader)
	if err != nil {
		return fmt.Errorf("writing file body: %w", err)
	}

	if written == 0 {
		return fmt.Errorf("no data was written to the output file")
	}

	if err = out.Sync(); err != nil {
		return fmt.Errorf("syncing file to disk: %w", err)
	}

	return
}
