package util

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// CloneRepo makes a shallow clone of the argument repo at a single ref.
//
// See https://stackoverflow.com/questions/3489173/how-to-clone-git-repository-with-specific-revision-changeset
// for a discussion of the commands run.
func CloneRepo(ctx context.Context, url string, ref string, path string) error {
	log.Debugf("Making clone of %s at ref=%s in %s", url, ref, path)

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	err = runGit(
		ctx,
		[]string{
			"init",
		},
		path,
	)
	if err != nil {
		return fmt.Errorf("Error running git init: %+v", err)
	}

	err = runGit(
		ctx,
		[]string{
			"remote",
			"add",
			"origin",
			url,
		},
		path,
	)
	if err != nil {
		return fmt.Errorf("Error updating git remote: %+v", err)
	}

	err = runGit(
		ctx,
		[]string{
			"fetch",
			"origin",
			"--depth",
			"1",
			ref,
		},
		path,
	)
	if err != nil {
		return fmt.Errorf("Error fetching ref: %+v", err)
	}

	err = runGit(
		ctx,
		[]string{
			"reset",
			"--hard",
			"FETCH_HEAD",
		},
		path,
	)
	if err != nil {
		return fmt.Errorf("Error resetting head: %+v", err)
	}

	return nil
}

func runGit(ctx context.Context, args []string, dir string) error {
	log.Debugf("Running git with args %+v", args)

	cmd := exec.CommandContext(
		ctx,
		"git",
		args...,
	)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error running git: %s", string(out))
	}

	return nil
}
