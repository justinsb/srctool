package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"os/exec"

	"k8s.io/klog/v2"
)

type ExecResult struct {
	Stdout string
	Stderr string

	ExitCode int
}

func (r *ExecResult) PrintOutput() {
	os.Stdout.Write([]byte(r.Stdout))
	os.Stderr.Write([]byte(r.Stderr))
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
	// fullPath, err := exec.LookPath("git")
	// if err != nil {
	// 	return nil, fmt.Errorf("unable to find git in path: %w", err)
	// }

	// argv := []string{fullPath}
	// argv = append(argv, args...)
	// attr := &os.ProcAttr{
	// 	Dir: dir,
	// 	Files: os.File{},
	// }
	// klog.V(1).Infof("running %s", strings.Join(argv, " "))

	// p, err := os.StartProcess(argv[0], argv, attr)
	// if err != nil {
	// 	return nil, fmt.Errorf("error running %q: %w", strings.Join(argv, " "), err)
	// }
	// state, err := p.Wait()
	// if err != nil {
	// 	return nil, fmt.Errorf("error running %q: %w", strings.Join(argv, " "), err)
	// }

	// result := &ExecResult{}

	// result.ExitCode = state.ExitCode()

	// if result.ExitCode != 0 {
	// 	err := fmt.Errorf("unexpected exit code %d", result.ExitCode)
	// 	err = fmt.Errorf("error running %q: %w", strings.Join(argv, " "), err)
	// 	return result, err
	// }

	// return result, nil

	cmd := exec.CommandContext(ctx, "git", args...)

	cmd.Dir = dir

	cmd.Stdin = os.Stdin
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
