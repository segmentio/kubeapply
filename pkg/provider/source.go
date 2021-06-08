package provider

import (
	"context"
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
	gitClientObj gitClient
}

func newSourceFetcher(gitClientObj gitClient) (*sourceFetcher, error) {
	return &sourceFetcher{
		gitClientObj: gitClientObj,
	}, nil
}

func (s *sourceFetcher) get(ctx context.Context, source string, dest string) error {
	matches := githubSourceRegexp.FindStringSubmatch(source)

	if len(matches) > 0 {
		repoURL := matches[1]
		path := matches[2]
		ref := matches[3]

		log.Infof("Cloning repo with source %s", source)

		cloneDir, err := ioutil.TempDir("", "kubeapply_clone")
		if err != nil {
			return err
		}
		defer os.RemoveAll(cloneDir)

		err = s.gitClientObj.cloneRepo(
			ctx,
			repoURL,
			ref,
			cloneDir,
		)
		if err != nil {
			return err
		}

		sourcePath := filepath.Join(cloneDir, path)
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
	return nil
}
