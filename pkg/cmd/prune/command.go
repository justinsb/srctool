package prune

import (
	"context"
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

	upstream, err := repo.FindRemoteTargetForPullRequests(ctx)
	if err != nil {
		return err
	}

	if err := repo.Fetch(ctx, upstream); err != nil {
		return err
	}

	allBranches, err := upstream.ListBranches(ctx)
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

	var pruneBranches []string

	for _, releaseBranch := range releaseBranches {
		result, err := repo.ExecGit(ctx, "branch", "--merged", releaseBranch.Name)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(result.Stdout, "\n") {
			tokens := strings.Fields(line)
			if len(tokens) == 0 {
				// Ignore empty lines
			} else if len(tokens) == 2 && tokens[0] == "*" {
				klog.Infof("skipping current branch %q", tokens[1])
			} else if len(tokens) == 1 {
				branchName := tokens[0]
				if _, found := releaseBranches[branchName]; found {
					klog.Infof("won't delete release branch %q", branchName)
				} else {
					pruneBranches = append(pruneBranches, branchName)
					klog.Infof("branch %q is merged into %q", branchName, releaseBranch.Name)
				}
			} else {
				return fmt.Errorf("cannot interpret branch line %q", line)
			}
		}
	}

	if !opt.DryRun {
		for _, pruneBranch := range pruneBranches {
			if err := repo.DeleteBranch(ctx, pruneBranch); err != nil {
				return err
			}
		}
	}

	return nil
}
