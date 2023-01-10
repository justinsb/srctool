package forks

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

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

	forkRemoteName := config["gitflow.fork.remote"]
	if forkRemoteName == "" {
		forkPrefix := "https://github.com/" + githubUsername + "/"
		var candidates []*git.Remote
		for _, remote := range remotes {
			if strings.HasPrefix(remote.FetchURL, forkPrefix) {
				candidates = append(candidates, remote)
			}
		}

		if len(candidates) == 0 {
			// TODO: Call gh repo fork?
			return fmt.Errorf("unable to find local fork (starting with %q)", forkPrefix)
		}
		if len(candidates) > 1 {
			return fmt.Errorf("found multiple local forks (starting with %q)", forkPrefix)
		}

		forkRemote := candidates[0]
		if forkRemote.Name != githubUsername {
			if err := forkRemote.Rename(ctx, githubUsername); err != nil {
				return err
			}
		}
		forkRemoteName = forkRemote.Name

		repoName := strings.TrimPrefix(forkRemote.FetchURL, forkPrefix)
		pushURL := "git@github.com:" + githubUsername + "/" + repoName
		if forkRemote.PushURL != pushURL {
			if err := forkRemote.UpdateURLs(ctx, forkRemote.FetchURL, pushURL); err != nil {
				return err
			}
		}

		if err := repo.SetConfig(ctx, "gitflow.fork.remote", forkRemoteName); err != nil {
			return err
		}
	}

	upstreamRemoteName := config["gitflow.upstream.remote"]
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
