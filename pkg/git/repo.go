package git

import (
	"context"
	"fmt"
	"os"
	"strings"

	git "github.com/libgit2/git2go/v31"
)

type Repo struct {
	Dir string

	gitRepo *git.Repository
}

func (r *Repo) Close() error {
	if r.gitRepo != nil {
		r.gitRepo.Free()
		r.gitRepo = nil
	}
	return nil
}

func OpenRepo(ctx context.Context) (*Repo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}

	// TODO: Find root?
	p := cwd
	flag := git.RepositoryOpenFlag(0) // Maybe RepositoryOpenNoSearch ?
	ceiling := ""                     // TODO:??
	repo, err := git.OpenRepositoryExtended(p, flag, ceiling)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repo %q: %w", p, err)
	}
	return &Repo{Dir: cwd, gitRepo: repo}, nil
}

func (r *Repo) GetRemote(ctx context.Context, remoteName string) (*Remote, error) {
	info, err := r.gitRepo.Remotes.Lookup(remoteName)
	if err != nil {
		return nil, fmt.Errorf("error looking up remote %q: %w", remoteName, err)
	}

	url := info.Url()
	return &Remote{
		repo: r,
		Name: remoteName,
		URL:  url,
	}, nil
}

func (r *Repo) FindRemoteTargetForPullRequests(ctx context.Context) (*Remote, error) {
	config, err := r.gitRepo.Config()
	if err != nil {
		return nil, fmt.Errorf("error getting repo config: %w", err)
	}
	key := "gitflow.upstream.remote"
	remote, err := config.LookupString(key)
	if err != nil {
		if git.IsErrorCode(err, git.ErrorCodeNotFound) {
			remote = ""
		} else {
			return nil, fmt.Errorf("error looking up config value %q: %w", key, err)
		}
	}
	if remote != "" {
		return r.GetRemote(ctx, remote)
	}

	remoteNames, err := r.gitRepo.Remotes.List()
	if err != nil {
		return nil, fmt.Errorf("error listing remotes: %w", err)
	}

	var candidates []*Remote
	for _, remoteName := range remoteNames {
		remote, err := r.GetRemote(ctx, remoteName)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, remote)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("cannot determine any target remote for pull requests")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	names := collect(candidates, func(b *Remote) string { return b.Name })
	return nil, fmt.Errorf("cannot determine unique target remote for pull requests, consider setting %q (candidates: %v)", key, strings.Join(names, ","))
}

func (r *Repo) FindUpstreamBranch(ctx context.Context) (*Branch, error) {
	upstreamRemote, err := r.FindRemoteTargetForPullRequests(ctx)
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

func (r *Repo) Fetch(ctx context.Context, remote *Remote) error {
	result, err := r.ExecGit(ctx, "fetch", remote.Name)
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return err
	}
	return nil
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
