package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"os/exec"

	"k8s.io/klog"
)

type ExecResult struct {
	Stdout string
	Stderr string

	ExitCode int
}

func execGit(ctx context.Context, dir string, args ...string) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)

	cmd.Dir = dir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	klog.V(1).Infof("running %s", strings.Join(cmd.Args, " "))

	err := cmd.Run()

	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitError.ExitCode()
	}

	if err != nil {
		err = fmt.Errorf("error running %q: %w", strings.Join(cmd.Args, " "), err)
	}
	return result, err
}

func execGitInteractive(ctx context.Context, dir string, args ...string) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)

	cmd.Dir = dir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	klog.V(1).Infof("running %s", strings.Join(cmd.Args, " "))

	err := cmd.Run()

	result := &ExecResult{}

	if exitError, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitError.ExitCode()
	}

	if err != nil {
		err = fmt.Errorf("error running %q: %w", strings.Join(cmd.Args, " "), err)
	}

	return result, err
}
