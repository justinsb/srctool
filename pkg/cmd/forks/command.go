package forks

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

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

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	parent.AddCommand(cmd)
}

type Options struct {
}

func (o *Options) InitDefaults() {

}

type Repo interface {
}

type GithubRepo struct {
	Organization string
	Repository   string
}

func ParseRepoFromURL(ctx context.Context, s string) Repo {
	if strings.HasPrefix(s, "https://github.com/") {
		u, err := url.Parse(s)
		if err != nil {
			klog.Warningf("unable to parse url %q: %v", s, err)
			return nil
		}
		pathTokens := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(pathTokens) == 2 {
			return &GithubRepo{
				Organization: pathTokens[0],
				Repository:   pathTokens[1],
			}
		}
	}

	if strings.HasPrefix(s, "git@github.com:") {
		s = strings.TrimPrefix(s, "git@github.com:")
		s = strings.TrimSuffix(s, ".git")

		tokens := strings.Split(strings.Trim(s, "/"), "/")
		if len(tokens) == 2 {
			return &GithubRepo{
				Organization: tokens[0],
				Repository:   tokens[1],
			}
		}
	}

	klog.Warningf("unknown repo %q", s)
	return nil
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

	githubUsername := os.Getenv("USER")

	forkRemoteName := config.Get("gitflow.fork.remote")
	if forkRemoteName == "" {
		var candidates []*git.Remote
		for _, remote := range remotes {
			repo := ParseRepoFromURL(ctx, remote.FetchURL)
			switch repo := repo.(type) {
			case *GithubRepo:
				if repo.Organization == githubUsername {
					candidates = append(candidates, remote)
				}
			}
		}

		if len(candidates) == 0 {
			// TODO: Call gh repo fork?
			return fmt.Errorf("unable to find local fork")
		}
		if len(candidates) > 1 {
			return fmt.Errorf("found multiple local forks")
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
		repoInfo := ParseRepoFromURL(ctx, forkRemote.FetchURL)
		pushURL := ""
		fetchURL := ""

		switch repoInfo := repoInfo.(type) {
		case *GithubRepo:
			if repoInfo.Organization == githubUsername {
				pushURL = "git@github.com:" + repoInfo.Organization + "/" + repoInfo.Repository
				fetchURL = "https://github.com/" + repoInfo.Organization + "/" + repoInfo.Repository
			}
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
			return fmt.Errorf("unable to find upstream remote")
		}
		if len(candidates) > 1 {
			return fmt.Errorf("found multiple upstream remotes")
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

	return nil
}
