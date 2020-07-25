package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// FileExists returns whether the given path exists and is a file.
func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return !info.IsDir(), nil
}

// DirExists returns whether the given path exists and is a directory.
func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

// RecursiveCopy does a recursive copy from the source directory to the destination one.
func RecursiveCopy(srcDir string, destDir string) error {
	ok, err := DirExists(destDir)
	if err != nil {
		return err
	}
	if !ok {
		info, err := os.Stat(srcDir)
		if err != nil {
			return err
		}

		err = os.MkdirAll(destDir, info.Mode())
		if err != nil {
			return err
		}
	}

	return filepath.Walk(
		srcDir,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(srcDir, subPath)
			if err != nil {
				return err
			}
			destPath := filepath.Join(destDir, relPath)

			if info.IsDir() {
				return os.MkdirAll(destPath, info.Mode())
			}

			toCopyFile := toCopy{
				src:     subPath,
				srcMode: info.Mode(),
				dest:    destPath,
			}
			return toCopyFile.copy()
		},
	)
}

type toCopy struct {
	src     string
	srcMode os.FileMode
	dest    string
}

func (t toCopy) copy() error {
	source, err := os.Open(t.src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(t.dest)
	if err != nil {
		return err
	}
	if err := dest.Chmod(t.srcMode); err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// RemoveDirs removes all directories that contain a file with the specified
// indicatorName (e.g., ".noexpand"). This is used to prune directories that
// are used for templating/generation but are not valid kube configs.
func RemoveDirs(rootDir string, indicatorName string) error {
	dirsToRemove := []string{}

	err := filepath.Walk(
		rootDir,
		func(subPath string, info os.FileInfo, err error) error {
			if !info.IsDir() && strings.ToLower(filepath.Base(subPath)) == indicatorName {
				dirsToRemove = append(dirsToRemove, filepath.Dir(subPath))
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	for _, dir := range dirsToRemove {
		log.Infof("Removing directory %s", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}

	return nil
}
