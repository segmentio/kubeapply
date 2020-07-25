package pullreq

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/segmentio/kubeapply/data"
	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/config"
)

var templates *template.Template

func init() {
	var err error
	templates, err = loadTemplates()
	if err != nil {
		panic(err)
	}
}

// ApplyCommentData stores data for templating out a "kubeapply apply" comment result.
type ApplyCommentData struct {
	ClusterApplies    []ClusterApply
	PullRequestClient PullRequestClient
	Env               string
}

type ClusterApply struct {
	ClusterConfig *config.ClusterConfig
	Results       []apply.Result
}

func (c ClusterApply) NumUpdates() int {
	updates := 0

	for _, result := range c.Results {
		if result.IsCreated() || result.IsUpdated() {
			updates++
		}
	}

	return updates
}

// FormatApplyComment generates the body of an apply comment result.
func FormatApplyComment(commentData ApplyCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"apply_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

// DiffCommentData stores data for templating out a "kubeapply diff" comment result.
type DiffCommentData struct {
	ClusterDiffs      []ClusterDiff
	PullRequestClient PullRequestClient
	Env               string
}

type ClusterDiff struct {
	ClusterConfig *config.ClusterConfig
	RawDiffs      string
}

func (c ClusterDiff) NumDiffs() int {
	if strings.TrimSpace(c.RawDiffs) == "" {
		return 0
	}

	count := 0

	// Look for "+++/---" line pairs
	//
	// TODO: Make diff outputs more structured so we can do this more precisely.
	lines := strings.Split(c.RawDiffs, "\n")

	for l := 1; l < len(lines); l++ {
		if strings.HasPrefix(lines[l], "+++ ") &&
			strings.HasPrefix(lines[l-1], "--- ") {
			count++
		}
	}

	return count
}

// FormatDiffComment generates the body of a diff comment result.
func FormatDiffComment(commentData DiffCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"diff_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

// ErrorCommentData represents data for templating out a kubeapply error comment result.
type ErrorCommentData struct {
	Error error
	Env   string
}

// FormatErrorComment generates the body of an error comment result.
func FormatErrorComment(commentData ErrorCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"error_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

// HelpCommentData stores data for templating out a "kubeapply help" comment result.
type HelpCommentData struct {
	ClusterConfigs []*config.ClusterConfig
	Env            string
}

// FormatHelpComment generates the body of a help comment result.
func FormatHelpComment(commentData HelpCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"help_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return string(out.Bytes()), nil
}

// StatusCommentData stores data for templating out a "kubeapply status" comment result.
type StatusCommentData struct {
	ClusterStatuses   []ClusterStatus
	PullRequestClient PullRequestClient
	Env               string
}

type ClusterStatus struct {
	ClusterConfig *config.ClusterConfig
	HealthSummary string
}

// FormatStatusComment generates the body of a status comment result.
func FormatStatusComment(commentData StatusCommentData) (string, error) {
	out := &bytes.Buffer{}

	err := templates.ExecuteTemplate(
		out,
		"status_comment.gotpl",
		commentData,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out.Bytes())), nil
}

func loadTemplates() (*template.Template, error) {
	templates := template.New("base")

	tempDir, err := ioutil.TempDir("", "templates")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	data.RestoreAssets(tempDir, "pkg/pullreq/templates")
	return templates.ParseGlob(
		filepath.Join(tempDir, "pkg/pullreq/templates/*.gotpl"),
	)
}
