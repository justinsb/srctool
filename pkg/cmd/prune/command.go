package prune

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use: "prune",
	}
	var opt Options
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	cmd.Flags().BoolVar(&opt.DryRun, "dry-run", opt.DryRun, "preview only, don't make changes")
	parent.AddCommand(cmd)
}

type Options struct {
	DryRun bool
}

func Run(ctx context.Context, opt Options) error {
	repo, err := git.OpenRepo(ctx)
	if err != nil {
		return err
	}
	defer repo.Close()

	upstreamRemote, err := repo.FindUpstreamRemoteForPullRequests(ctx)
	if err != nil {
		return err
	}

	if err := upstreamRemote.Fetch(ctx); err != nil {
		return err
	}

	allBranches, err := upstreamRemote.ListBranches(ctx)
	if err != nil {
		return err
	}

	releaseBranches := make(map[string]*git.Branch)
	for _, branch := range allBranches {
		if branch.ShortName == "main" || branch.ShortName == "master" {
			releaseBranches[branch.ShortName] = branch
		} else if strings.HasPrefix(branch.ShortName, "release-") {
			releaseBranches[branch.ShortName] = branch
		}
	}

	if len(releaseBranches) == 0 {
		return fmt.Errorf("cannot determine any release branches")
	}

	fmt.Printf("checking for branches merged into any of %v\n", mapValues(releaseBranches))

	pruneBranches := make(map[string]bool)

	for _, releaseBranch := range releaseBranches {
		args := []string{"branch", "--merged", releaseBranch.Name}
		result, err := repo.ExecGit(ctx, args...)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(result.Stdout, "\n") {
			tokens := strings.Fields(line)

			branchName := ""
			if len(tokens) == 0 {
				// Ignore empty lines
			} else if len(tokens) == 2 && tokens[0] == "+" {
				// Checked out as worktree somewhere
				branchName = tokens[1]
			} else if len(tokens) == 2 && tokens[0] == "*" {
				// Current branch
				klog.Infof("skipping current branch %q", tokens[1])
			} else if len(tokens) == 1 {
				branchName = tokens[0]
			} else {
				return fmt.Errorf("cannot interpret branch line %q (from command git %s)", line, strings.Join(args, " "))
			}

			if branchName != "" {
				if _, found := releaseBranches[branchName]; found {
					klog.Infof("won't delete release branch %q", branchName)
				} else {
					pruneBranches[branchName] = true
					klog.Infof("branch %q is merged into %q", branchName, releaseBranch.Name)
				}
			}
		}
	}

	if !opt.DryRun {
		var errs []error
		for pruneBranch := range pruneBranches {
			if err := repo.DeleteBranch(ctx, pruneBranch); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	return nil
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	var keys []K
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func mapValues[K comparable, V any](m map[K]V) []V {
	var values []V
	for _, v := range m {
		values = append(values, v)
	}
	return values
}
