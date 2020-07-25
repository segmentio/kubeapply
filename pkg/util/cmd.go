package util

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Printer is a function that prints out a string, e.g. to stdout.
type Printer func(input string)

// LogrusInfoPrinter returns a printer that prints via logrus at the info level.
func LogrusInfoPrinter(prefix string) Printer {
	return func(input string) {
		log.Infof("%s %s", prefix, input)
	}
}

// LogrusWarnPrinter returns a printer that prints via logrus at the warn level.
func LogrusWarnPrinter(prefix string) Printer {
	return func(input string) {
		log.Warnf("%s %s", prefix, input)
	}
}

// LogrusDebugPrinter returns a printer that prints via logrus at the debug level.
func LogrusDebugPrinter(prefix string) Printer {
	return func(input string) {
		log.Debugf("%s %s", prefix, input)
	}
}

// RunCmdWithPrinters runs a command with output streamed via custom printer functions.
// Adapted from example in https://github.com/golang/go/issues/19685#issuecomment-288949629.
func RunCmdWithPrinters(
	ctx context.Context,
	command string,
	args []string,
	extraEnv []string,
	blockedEnv map[string]struct{},
	stdoutPrinter Printer,
	stderrPrinter Printer,
) error {
	log.Debugf("Running %s with args %+v", command, args)

	cmd := exec.CommandContext(ctx, command, args...)
	envVars := os.Environ()

	cmdEnvVars := []string{}

	for _, envVar := range envVars {
		components := strings.SplitN(envVar, "=", 2)
		if _, ok := blockedEnv[components[0]]; !ok {
			cmdEnvVars = append(cmdEnvVars, envVar)
		}
	}

	cmdEnvVars = append(
		cmdEnvVars,
		extraEnv...,
	)

	cmd.Env = cmdEnvVars
	cmd.Stdin = os.Stdin

	done := make(chan struct{}, 2)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	go func() {
		for stdoutScanner.Scan() {
			stdoutPrinter(stdoutScanner.Text())
		}
		done <- struct{}{}
	}()

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stderrScanner := bufio.NewScanner(stderrPipe)
	go func() {
		for stderrScanner.Scan() {
			stderrPrinter(stderrScanner.Text())
		}
		done <- struct{}{}
	}()

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Wait for all scanner goroutines to finish
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			break
		}
	}

	return cmd.Wait()
}
