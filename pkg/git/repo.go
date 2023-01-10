package git

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/klog/v2"
)

type Repo struct {
	Dir string

	config map[string]string
}

func (r *Repo) Close() error {
	return nil
}

func OpenRepo(ctx context.Context) (*Repo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}

	// TODO: Find root?
	p := cwd

	r := &Repo{Dir: cwd}
	// We list config as a quick check that this is a real git directory
	config, err := r.ListConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repo %q: %w", p, err)
	}
	return &Repo{Dir: cwd, config: config}, nil
}

func (r *Repo) GetRemote(ctx context.Context, remoteName string) (*Remote, error) {
	remotes, err := r.ListRemotes(ctx)
	if err != nil {
		return nil, err
	}
	remote := remotes[remoteName]
	if remote == nil {
		return nil, fmt.Errorf("remote %q not found", remoteName)
	}
	return remote, nil
}

func (r *Repo) ListConfig(ctx context.Context) (map[string]string, error) {
	if r.config != nil {
		return r.config, nil
	}

	result, err := r.ExecGit(ctx, "config", "--list")
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return nil, err
	}

	config := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error parsing output: %w", err)
			}
			break
		}

		line := scanner.Text()
		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("error parsing line %q (expected 2 tokens)", line)
		}

		k := tokens[0]
		v := tokens[1]
		if config[k] != "" {
			return nil, fmt.Errorf("found duplicate config key %q", k)
		}
		config[k] = v
	}

	r.config = config
	return config, nil
}

func (r *Repo) SetConfig(ctx context.Context, k, v string) error {
	result, err := r.ExecGit(ctx, "config", k, v)
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return err
	}

	return nil
}

func (r *Repo) ListRemotes(ctx context.Context) (map[string]*Remote, error) {
	result, err := r.ExecGit(ctx, "remote", "-v")
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return nil, err
	}

	remotes := make(map[string]*Remote)
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error parsing output: %w", err)
			}
			break
		}

		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) != 3 {
			return nil, fmt.Errorf("error parsing line %q (expected 3 tokens)", line)
		}

		name := tokens[0]
		remote := remotes[name]
		if remote == nil {
			remote = &Remote{Name: name, repo: r}
			remotes[name] = remote
		}
		switch tokens[2] {
		case "(fetch)":
			if remote.FetchURL != "" {
				// TODO: This may be allowed by git
				return nil, fmt.Errorf("found multiple fetch urls for remote %q", name)
			}
			remote.FetchURL = tokens[1]
		case "(push)":
			if remote.PushURL != "" {
				// TODO: This may be allowed by git
				return nil, fmt.Errorf("found multiple push urls for remote %q", name)
			}
			remote.PushURL = tokens[1]

		default:
			return nil, fmt.Errorf("error parsing line %q (expected push or fetch)", line)
		}
	}

	return remotes, nil
}

func (r *Repo) FindUpstreamRemoteForPullRequests(ctx context.Context) (*Remote, error) {
	config, err := r.ListConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting repo config: %w", err)
	}

	key := "gitflow.upstream.remote"
	remote := config[key]
	if remote != "" {
		return r.GetRemote(ctx, remote)
	}

	remotes, err := r.ListRemotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing remotes: %w", err)
	}

	var candidates []*Remote
	for _, remote := range remotes {
		candidates = append(candidates, remote)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("cannot determine any upstream remote for pull requests")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	names := collect(candidates, func(b *Remote) string { return b.Name })
	return nil, fmt.Errorf("cannot determine unique upstream remote for pull requests, consider setting %q (candidates: %v)", key, strings.Join(names, ","))
}

func (r *Repo) FindForkRemoteForPullRequests(ctx context.Context) (*Remote, error) {
	config, err := r.ListConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting repo config: %w", err)
	}
	key := "gitflow.fork.remote"
	remote := config[key]
	if remote != "" {
		return r.GetRemote(ctx, remote)
	}

	remotes, err := r.ListRemotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing remotes: %w", err)
	}

	var candidates []*Remote
	for _, remote := range remotes {
		// TODO: Match os.Getenv("USER")?  Match "fork"?
		candidates = append(candidates, remote)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("cannot determine any fork remote for pull requests")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	names := collect(candidates, func(b *Remote) string { return b.Name })
	return nil, fmt.Errorf("cannot determine unique fork remote for pull requests, consider setting %q (candidates: %v)", key, strings.Join(names, ","))
}

func (r *Repo) CurrentBranch(ctx context.Context) (*Branch, error) {
	result, err := r.ExecGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(result.Stdout)
	if name == "" {
		return nil, fmt.Errorf("cannot find current branch (stdout was %q, stderr was %q)", result.Stdout, result.Stderr)
	}
	return &Branch{Name: name, ShortName: name}, nil
}

// TODO: Maybe put this on a workdir object?
func (r *Repo) CheckoutNewBranch(ctx context.Context, newBranchName string, fromBranch *Branch) (*Branch, error) {
	_, err := r.ExecGit(ctx, "checkout", "-b", newBranchName, fromBranch.Name)
	if err != nil {
		return nil, err
	}
	return &Branch{Name: newBranchName, ShortName: newBranchName}, nil
}

// TODO: Maybe put this on a workdir object?
func (r *Repo) Checkout(ctx context.Context, branch *Branch) error {
	_, err := r.ExecGit(ctx, "checkout", branch.Name)
	if err != nil {
		return err
	}
	return nil
}

// TODO: Maybe put this on a workdir object?
func (r *Repo) CherryPick(ctx context.Context, shas []string) error {
	args := []string{"cherry-pick"}
	args = append(args, shas...)
	_, err := r.ExecGit(ctx, args...)
	if err != nil {
		return err
	}
	return nil
}

type PushOptions struct {
	SetUpstream bool
}

// TODO: Maybe put this on a workdir object?
func (r *Repo) Push(ctx context.Context, remote *Remote, opt PushOptions) error {
	args := []string{"push"}
	if opt.SetUpstream {
		args = append(args, "--set-upstream")
	}
	args = append(args, remote.Name)

	result, err := r.ExecGit(ctx, args...)
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}
		return err
	}
	return nil
}

func (r *Repo) FindUpstreamBranch(ctx context.Context) (*Branch, error) {
	log := klog.FromContext(ctx)
	upstreamRemote, err := r.FindUpstreamRemoteForPullRequests(ctx)
	if err != nil {
		return nil, err
	}

	branches, err := upstreamRemote.ListBranches(ctx)
	if err != nil {
		return nil, err
	}

	var candidates []*Branch
	for _, branch := range branches {
		isMain := false
		switch branch.ShortName {
		case "main", "master":
			isMain = true
		}
		if !isMain {
			continue
		}
		candidates = append(candidates, branch)
	}

	if len(candidates) == 0 {
		log.Info("branches", "branches", branches)
		return nil, fmt.Errorf("cannot determine (any) upstream branch")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	names := collect(candidates, func(b *Branch) string { return b.Name })
	return nil, fmt.Errorf("cannot determine unique upstream branch (candidates: %v)", strings.Join(names, ","))
}

func collect[T any, V any](items []T, mapper func(t T) V) []V {
	var values []V
	for i := range items {
		v := mapper(items[i])
		values = append(values, v)
	}
	return values
}

func (r *Repo) ExecGit(ctx context.Context, args ...string) (*ExecResult, error) {
	return execGit(ctx, r.Dir, args...)
}

func (r *Repo) ExecGitInteractive(ctx context.Context, args ...string) (*ExecResult, error) {
	return execGitInteractive(ctx, r.Dir, args...)
}

// type RebaseOptions struct {
// 	Autosquash bool
// 	Upstream   *Branch
// }

// func (r *Repo) Rebase(ctx context.Context, options RebaseOptions) error {
// 	args := []string{"rebase"}
// 	if options.Autosquash {
// 		args = append(args, "--autosquash")
// 	}
// 	args = append(args, options.Upstream.Name)
// 	_, err := r.execGit(ctx, args...)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
