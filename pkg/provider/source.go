package provider

import (
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/segmentio/kubeapply/pkg/util"
	log "github.com/sirupsen/logrus"
)

var (
	githubSourceRegexp = regexp.MustCompile(
		"^(git@github.com:[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+)//([a-zA-Z0-9_/-]*)[?]ref=([a-zA-Z0-9_/.-]+)$",
	)
)

type sourceFetcher struct {
	cacheDir     string
	gitClientObj gitClient
}

func newSourceFetcher(gitClientObj gitClient) (*sourceFetcher, error) {
	tempDir, err := ioutil.TempDir("", "sources")
	if err != nil {
		return nil, err
	}

	return &sourceFetcher{
		cacheDir:     tempDir,
		gitClientObj: gitClientObj,
	}, nil
}

func (s *sourceFetcher) get(ctx context.Context, source string, dest string) error {
	matches := githubSourceRegexp.FindStringSubmatch(source)

	if len(matches) > 0 {
		repoURL := matches[1]
		path := matches[2]
		ref := matches[3]

		hash := md5.New()
		hash.Write([]byte(source))
		hashSum := fmt.Sprintf("%x", hash.Sum(nil))
		sourceRoot := filepath.Join(s.cacheDir, hashSum)

		if ok, _ := util.DirExists(sourceRoot); !ok {
			log.Infof("Cloning repo with source %s", source)

			err := s.gitClientObj.cloneRepo(
				ctx,
				repoURL,
				ref,
				sourceRoot,
			)
			if err != nil {
				return err
			}
		} else {
			log.Infof("Found source %s (hash=%s) in cache", source, hashSum)
		}

		sourcePath := filepath.Join(sourceRoot, path)

		if err := util.RecursiveCopy(sourcePath, dest); err != nil {
			return err
		}
	} else {
		log.Debugf("Copying profile from %s to %s", source, dest)
		if err := util.RecursiveCopy(source, dest); err != nil {
			return err
		}
	}

	return nil
}

func (s *sourceFetcher) cleanup() error {
	return os.RemoveAll(s.cacheDir)
}
