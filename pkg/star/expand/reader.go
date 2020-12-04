package expand

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stripe/skycfg"
)

var urlRegex = regexp.MustCompile("^([a-zA-Z0-9._-]+)://(.*)$")

type urlFileReader struct {
	root     string
	repoDirs map[repoRef]string
	tempDir  string
}

type repoRef struct {
	url string
	ref string
}

func (r repoRef) dirName() string {
	// Convert the repo ref to a nice-looking directory name
	dirName := fmt.Sprintf("%s__ref_%s", r.url, r.ref)
	dirName = strings.Replace(dirName, "/", "_", -1)
	dirName = strings.Replace(dirName, ":", "_", -1)
	dirName = strings.Replace(dirName, "@", "_", -1)

	return dirName
}

var _ skycfg.FileReader = (*urlFileReader)(nil)

// NewURLFileReader returns a skycfg FileReader that reads files from local disk, resolving
// remote URLs if needed.
func NewURLFileReader(root string) (*urlFileReader, error) {
	tempDir, err := ioutil.TempDir("", "files")
	if err != nil {
		return nil, err
	}

	return &urlFileReader{
		root:    root,
		tempDir: tempDir,
	}, nil
}

// Resolve converts a skycfg import path to a local file path.
func (r *urlFileReader) Resolve(
	ctx context.Context,
	name string,
	fromPath string,
) (string, error) {
	if fromPath == "" {
		return name, nil
	}

	matches := urlRegex.FindStringSubmatch(name)
	if len(matches) != 3 {
		// Try assuming a file
		matches = urlRegex.FindStringSubmatch(fmt.Sprintf("file://%s", name))
		if len(matches) != 3 {
			return "", fmt.Errorf("Invalid URL: %s", name)
		}
	}

	scheme := matches[1]
	remainder := matches[2]

	switch scheme {
	case "file":
		if strings.HasPrefix(remainder, "//") {
			// Treat path as being relative to cluster root
			return filepath.Join(
				r.root,
				remainder[2:],
			), nil
		}

		// Treat path as being relative to this file
		return filepath.Join(
			filepath.Dir(fromPath),
			remainder,
		), nil
	case "git", "git-https":
		ref := repoRef{}

		refIndex := strings.Index(remainder, "?ref=")
		if refIndex >= 0 {
			// Strip off ref
			ref.ref = remainder[(refIndex + len("?ref=")):]
			remainder = remainder[:refIndex]
		} else {
			ref.ref = "master"
		}

		subComponents := strings.Split(remainder, "//")

		if len(subComponents) != 2 {
			return "", fmt.Errorf(
				"Expected git URL to be in format [repo]//[path]: %s",
				remainder,
			)
		}
		ref.url = subComponents[0]
		subPath := subComponents[1]

		if scheme == "git-https" {
			// Add an https in front of the url in the git-https cases
			ref.url = fmt.Sprintf("https://%s", ref.url)
		}

		repoDir, ok := r.repoDirs[ref]
		if !ok {
			repoDir = filepath.Join(r.tempDir, ref.dirName())
			err := util.CloneRepo(
				ctx,
				ref.url,
				ref.ref,
				repoDir,
			)
			if err != nil {
				return "", err
			}
		}

		return filepath.Join(repoDir, subPath), nil
	default:
		return "", fmt.Errorf("Unresolveable name: %s", name)
	}
}

// ReadFile returns the contents of the file at the argument path.
func (r *urlFileReader) ReadFile(
	ctx context.Context,
	path string,
) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (r *urlFileReader) Close() error {
	return os.RemoveAll(r.tempDir)
}
