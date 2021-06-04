package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/segmentio/kubeapply/pkg/util"
)

type gitClient interface {
	cloneRepo(ctx context.Context, repoURL string, ref string, dest string) error
}

type commandLineGitClient struct{}

var _ gitClient = (*commandLineGitClient)(nil)

func (c *commandLineGitClient) cloneRepo(
	ctx context.Context,
	repoURL string,
	ref string,
	dest string,
) error {
	return util.CloneRepo(ctx, repoURL, ref, dest)
}

type fakeGitClient struct {
	calls []string
}

var _ gitClient = (*fakeGitClient)(nil)

func (f *fakeGitClient) cloneRepo(
	ctx context.Context,
	repoURL string,
	ref string,
	dest string,
) error {
	f.calls = append(
		f.calls,
		fmt.Sprintf("cloneRepo repoURL=%s,ref=%s", repoURL, ref),
	)
	profilePath := filepath.Join(dest, "profiles", "profile.txt")

	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}

	file, err := os.Create(profilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("repoURL=%s ref=%s", repoURL, ref))
	return err
}
