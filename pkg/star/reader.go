package star

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/stripe/skycfg"
)

type localFileReader struct {
	root string
}

var _ skycfg.FileReader = (*localFileReader)(nil)

func NewLocalFileReader(root string) (*localFileReader, error) {
	return &localFileReader{
		root: root,
	}, nil
}

func (r *localFileReader) Resolve(
	ctx context.Context,
	name string,
	fromPath string,
) (string, error) {
	if fromPath == "" {
		return name, nil
	}

	var resolved string

	if strings.HasPrefix(name, "//") {
		// Treat path as being relative to cluster root
		resolved = filepath.Join(
			r.root,
			name[2:],
		)
	} else {
		// Treat path as being relative to this file
		resolved = filepath.Join(
			filepath.Dir(fromPath),
			name,
		)
	}

	return resolved, nil
}

func (r *localFileReader) ReadFile(
	ctx context.Context,
	path string,
) ([]byte, error) {
	return ioutil.ReadFile(path)
}
