package forks

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use: "forks",
	}
	var opt Options
	opt.InitDefaults()

	cmd.Flags().BoolVar(&opt.PushWithSSH, "push-with-ssh", opt.PushWithSSH, "configure forks to use SSH when pushing")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	parent.AddCommand(cmd)
}

type Options struct {
	// PushWithSSH controls whether we will rewrite the upstream to use SSH, when the remote is our own fork
	PushWithSSH bool
}

func (o *Options) InitDefaults() {

}

func Run(ctx context.Context, opt Options) error {
	repo, err := git.OpenRepo(ctx)
	if err != nil {
		return err
	}
	defer repo.Close()

	config, err := repo.ListConfig(ctx)
	if err != nil {
		return err
	}
	remotes, err := repo.ListRemotes(ctx)
	if err != nil {
		return err
	}

	githubUsername := os.Getenv("GITHUB_USER")
	if githubUsername == "" {
		githubUsername = os.Getenv("USER")
	}

	forkRemoteName := config.Get("gitflow.fork.remote")
	if forkRemoteName == "" {
		var candidates []*git.Remote
		for _, remote := range remotes {
			repo := git.ParseRepoFromURL(ctx, remote.FetchURL)
			switch repo := repo.(type) {
			case *git.GithubForgeInfo:
				if repo.Organization == githubUsername {
					candidates = append(candidates, remote)
				}
			default:
				klog.Warningf("unknown repo type %T", repo)
			}
		}

		if len(candidates) == 0 {
			// TODO: Call gh repo fork?
			return fmt.Errorf("found no candidates for your fork (username %q)", githubUsername)
		}
		if len(candidates) > 1 {
			return fmt.Errorf("found multiple candidates for your fork (username %q): %v", githubUsername, Map(candidates, func(r *git.Remote) string { return r.Name }))
		}

		forkRemote := candidates[0]
		if forkRemote.Name != githubUsername {
			if err := forkRemote.Rename(ctx, githubUsername); err != nil {
				return err
			}
		}
		forkRemoteName = forkRemote.Name

		// Configure explicitly
		if err := repo.SetConfig(ctx, "gitflow.fork.remote", forkRemoteName); err != nil {
			return err
		}

		// Refresh remotes
		remotes, err = repo.ListRemotes(ctx)
		if err != nil {
			return err
		}
	}

	// Fix urls on remote fork, if github
	forkRemote := remotes[forkRemoteName]
	if forkRemote == nil {
		return fmt.Errorf("remote fork %q not found", forkRemoteName)
	} else {
		repoInfo := git.ParseRepoFromURL(ctx, forkRemote.FetchURL)
		pushURL := ""
		fetchURL := ""

		switch repoInfo := repoInfo.(type) {
		case *git.GithubForgeInfo:
			if opt.PushWithSSH && repoInfo.Organization == githubUsername {
				pushURL = "git@github.com:" + repoInfo.Organization + "/" + repoInfo.Repository
				fetchURL = "https://github.com/" + repoInfo.Organization + "/" + repoInfo.Repository
			}
		default:
			klog.Infof("unknown repo type %T", repoInfo)
		}

		if pushURL != "" && fetchURL != "" {
			if forkRemote.PushURL != pushURL || forkRemote.FetchURL != fetchURL {
				if err := forkRemote.UpdateURLs(ctx, fetchURL, pushURL); err != nil {
					return err
				}
			}
		} else {
			klog.Warningf("cannot determine correct urls for %q", forkRemote.FetchURL)
		}
	}

	upstreamRemoteName := config.Get("gitflow.upstream.remote")
	if upstreamRemoteName == "" {
		expectedName := "upstream"

		var candidates []*git.Remote
		for _, remote := range remotes {
			if remote.Name == expectedName {
				candidates = append(candidates, remote)
			}
		}

		if len(candidates) == 0 {
			return fmt.Errorf("unable to find remote for upstream repo")
		}
		if len(candidates) > 1 {
			return fmt.Errorf("found multiple potential remotes for upstream repo")
		}

		upstreamRemote := candidates[0]
		if upstreamRemote.Name != expectedName {
			if err := upstreamRemote.Rename(ctx, expectedName); err != nil {
				return err
			}
		}
		upstreamRemoteName = upstreamRemote.Name

		if err := repo.SetConfig(ctx, "gitflow.upstream.remote", upstreamRemoteName); err != nil {
			return err
		}
	}

	// Fix urls on upstream fork, if github
	upstreamRemote := remotes[upstreamRemoteName]
	if upstreamRemote == nil {
		return fmt.Errorf("upstream remote %q not found", upstreamRemoteName)
	} else {
		repoInfo := git.ParseRepoFromURL(ctx, upstreamRemote.FetchURL)
		pushURL := ""
		fetchURL := ""

		switch repoInfo := repoInfo.(type) {
		case *git.GithubForgeInfo:
			fetchURL = "https://github.com/" + repoInfo.Organization + "/" + repoInfo.Repository
			pushURL = "nope"
		default:
			klog.Infof("unknown repo type %T", repoInfo)
		}

		if pushURL != "" && fetchURL != "" {
			if upstreamRemote.PushURL != pushURL || upstreamRemote.FetchURL != fetchURL {
				if err := upstreamRemote.UpdateURLs(ctx, fetchURL, pushURL); err != nil {
					return err
				}
			}
		} else {
			klog.Warningf("cannot determine correct urls for %q", upstreamRemote.FetchURL)
		}
	}

	return nil
}

func Map[T any, T2 any](in []T, fn func(t T) T2) []T2 {
	var out []T2
	for _, t := range in {
		out = append(out, fn(t))
	}
	return out
}
