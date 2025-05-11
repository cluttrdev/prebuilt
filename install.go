package main

import (
	"io"
	"os"
)

// Install copies the source file to the destination file
// and sets the destination file's permissions to `rwxr-x--x`.
func Install(src string, dst string) error {
	ifile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = ifile.Close()
	}()

	ofile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = ofile.Close()
	}()

	_, err = io.Copy(ofile, ifile)
	if err != nil {
		return err
	}

	if err := os.Chmod(dst, 0751); err != nil {
		return err
	}

	return nil
}
