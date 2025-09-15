package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Create a pull request",
	}
	var opt Options
	opt.InitDefaults()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt, args[0], args[1:])
	}
	parent.AddCommand(cmd)
}

type Options struct {
}

func (o *Options) InitDefaults() {

}

func Run(ctx context.Context, opt Options, prBranchName string, shas []string) error {
	repo, err := git.OpenRepo(ctx)
	if err != nil {
		return err
	}
	defer repo.Close()

	forkRemote, err := repo.FindForkRemoteForPullRequests(ctx)
	if err != nil {
		return err
	}

	upstream, err := repo.FindUpstreamBranch(ctx)
	if err != nil {
		return err
	}

	originalBranch, err := repo.CurrentBranch(ctx)
	if err != nil {
		return err
	}

	if err := upstream.Remote.Fetch(ctx); err != nil {
		return err
	}

	if _, err := repo.CheckoutNewBranch(ctx, prBranchName, upstream); err != nil {
		return err
	}

	if err := repo.CherryPick(ctx, shas); err != nil {
		return err
	}

	if err := repo.Push(ctx, forkRemote, git.PushOptions{SetUpstream: true}); err != nil {
		return err
	}

	{
		forkRemoteGithubName := "justinsb" // TODO: extract from https://github.com/justinsb/foo.git or git@github.com/justinsb/foo.git
		args := []string{"gh", "pr", "create", "--base", upstream.ShortName, "--head", forkRemoteGithubName + ":" + prBranchName, "--fill"}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error running %s: %w", args, err)
		}
	}

	if err := repo.Checkout(ctx, originalBranch); err != nil {
		return err
	}

	return nil
}
