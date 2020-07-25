package util

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

var (
	urlRegex   = regexp.MustCompile("^([a-zA-Z0-9._-]+)://(.*)$")
	s3URLRegex = regexp.MustCompile("^s3://([a-zA-Z0-9._-]+)/(.*)$")
)

// RestoreData generates a local version of the resource(s) at the argument URL. Currently, it
// supports the schemes "file://", "http://", "https://", "git://", "git-https://", and "s3://".
//
// If there is no scheme, then "file://" is assumed.
//
// In the http(s) and s3 cases, the url must refer to an archive. In the file case, it can
// refer to either an archive or a directory.
//
// The rootDir argument is used in the file case as the base for relative file URLs. It is
// unused in other cases.
func RestoreData(
	ctx context.Context,
	rootDir string,
	url string,
	destDir string,
) error {
	matches := urlRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		// Try assuming a file
		matches = urlRegex.FindStringSubmatch(fmt.Sprintf("file://%s", url))
		if len(matches) != 3 {
			return fmt.Errorf("Invalid URL: %s", url)
		}
	}

	scheme := matches[1]
	remainder := matches[2]

	isArchive := strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz")

	switch scheme {
	case "file":
		var absPath string

		if filepath.IsAbs(remainder) {
			absPath = remainder
		} else {
			absPath = filepath.Join(rootDir, remainder)
		}

		if isArchive {
			return unarchiveTarGz(ctx, absPath, destDir)
		}

		return RecursiveCopy(absPath, destDir)
	case "git", "git-https":
		var repoURL string

		if scheme == "git" {
			repoURL = remainder
		} else {
			// Add an https in front of the remainder in git-https case
			repoURL = fmt.Sprintf("https://%s", remainder)
		}

		refIndex := strings.Index(repoURL, "?ref=")
		var gitRef string

		if refIndex >= 0 {
			gitRef = repoURL[(refIndex + len("?ref=")):]
			repoURL = repoURL[:refIndex]
		} else {
			gitRef = "master"
		}

		return CloneRepo(
			ctx,
			repoURL,
			gitRef,
			destDir,
		)
	case "http", "https":
		log.Debugf("Getting http url: %s", url)

		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 300 {
			return fmt.Errorf("Non-200 response code from request: %d", resp.StatusCode)
		}

		defer resp.Body.Close()
		return unarchiveReader(ctx, resp.Body, destDir)
	case "s3":
		log.Debugf("Getting s3 archive: %s", url)

		bucket, key, err := ParseS3URL(url)
		if err != nil {
			return err
		}

		sess := session.Must(session.NewSession())
		s3Client := s3.New(sess)
		resp, err := s3Client.GetObjectWithContext(
			ctx,
			&s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			},
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return unarchiveReader(ctx, resp.Body, destDir)
	default:
		return fmt.Errorf("Unrecognized resource url: %s", url)
	}
}

// ParseURL splits an s3 url into its bucket and key
func ParseS3URL(url string) (string, string, error) {
	matches := s3URLRegex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("Invalid s3 url: %s", url)
	}

	return matches[1], matches[2], nil
}

func unarchiveReader(ctx context.Context, reader io.Reader, destDir string) error {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil
	}

	tempFile, err := ioutil.TempFile("", "output")
	if err != nil {
		return nil
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(contents)
	if err != nil {
		log.Printf("Error writing to tempfile: %+v", err)
		return err
	}

	err = tempFile.Close()
	if err != nil {
		log.Printf("Error closing tempfile: %+v", err)
		return nil
	}

	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return err
	}

	return unarchiveTarGz(ctx, tempFile.Name(), destDir)
}

func unarchiveTarGz(ctx context.Context, source string, dest string) error {
	tarArgs := []string{
		"-xzf",
		source,
		"-C",
		dest,
	}

	cmd := exec.CommandContext(ctx, "tar", tarArgs...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("Error running tar (%+v): %s", err, string(output))
	}

	return nil
}
