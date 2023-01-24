package cherry

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use: "cherry",
	}
	var opt Options
	opt.InitDefaults()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt, args[0])
	}
	parent.AddCommand(cmd)
}

type Options struct {
}

func (o *Options) InitDefaults() {

}

func Run(ctx context.Context, opt Options, prNumber string) error {

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

	targetBranch, err := repo.CurrentBranch(ctx)
	if err != nil {
		return err
	}

	prBranchName := "automated-cherry-pick-of-#" + prNumber + "-" + targetBranch.Name

	// if err := targetBranch.Remote.Fetch(ctx); err != nil {
	// 	return err
	// }

	if _, err := repo.CheckoutNewBranch(ctx, prBranchName, targetBranch); err != nil {
		return err
	}

	pr, err := upstream.Remote.GetPullRequest(ctx, prNumber)
	if err != nil {
		return err
	}

	if err := repo.CherryPick(ctx, pr.Commits()); err != nil {
		return err
	}

	if err := repo.Push(ctx, forkRemote, git.PushOptions{SetUpstream: true}); err != nil {
		return err
	}

	title := fmt.Sprintf("Automated cherry pick of #" + prNumber + ": " + pr.Title() + "\n")
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("Cherry pick of #" + prNumber + " on " + targetBranch.Name + "\n"))
	body.WriteString(fmt.Sprintf("\n"))
	body.WriteString(fmt.Sprintf("#" + prNumber + ":" + pr.Title() + "\n"))

	{
		forkRemoteGithubName := "justinsb" // TODO: extract from https://github.com/justinsb/foo.git or git@github.com/justinsb/foo.git
		args := []string{"gh", "pr", "create", "--base", targetBranch.ShortName, "--head", forkRemoteGithubName + ":" + prBranchName, "--title", title, "--body-file", "-"}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = &body

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error running %s: %w", args, err)
		}
	}

	if err := repo.Checkout(ctx, targetBranch); err != nil {
		return err
	}

	return nil
}
