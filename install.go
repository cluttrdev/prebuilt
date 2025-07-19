package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	dstDir := filepath.Dir(dst)
	dstName := filepath.Base(dst)

	// write src to new temporary dst
	dstNew := filepath.Join(dstDir, fmt.Sprintf(".%s.new", dstName))
	ofile, err := os.OpenFile(dstNew, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
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

	// close ofile here, since windows wouldn't let us move the new file
	_ = ofile.Close()

	if _, err := os.Stat(dst); err == nil { // file exists
		dstOld := filepath.Join(dstDir, fmt.Sprintf(".%s.old", dstName))

		// delete existing old file (for windows' sake)
		_ = os.Remove(dstOld)

		// move existing file
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}
	}

	// move the new file
	if err := os.Rename(dstNew, dst); err != nil {
		return err
	}

	return nil
}
