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
		Use:   "cherry <pr-number>",
		Short: "Cherry-pick a pull request from upstream to current branch",
		Long: `Cherry-pick a pull request from the upstream repository to the current branch.

This command will:
1. Create a new branch based on the current branch (or specified branch)
2. Cherry-pick all commits from the specified pull request
3. Push the new branch to your fork
4. Create a new pull request for the cherry-picked changes
5. Switch back to the original branch

The new branch will be named: automated-cherry-pick-of-#<pr-number>-<target-branch>
The new pull request will reference the original PR and include appropriate metadata.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Cherry-pick PR #1234 from upstream to current branch
  srctool cherry 1234

  # Cherry-pick PR #1234 from upstream to release-1.32 branch
  srctool cherry 1234 --branch release-1.32`,
	}
	var opt Options
	opt.InitDefaults()

	cmd.Flags().StringVar(&opt.Branch, "branch", "", "Target branch to cherry-pick to (defaults to current branch)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt, args[0])
	}
	parent.AddCommand(cmd)
}

type Options struct {
	Branch string
}

func (o *Options) InitDefaults() {
	o.Branch = ""
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

	var targetBranch *git.Branch
	if opt.Branch != "" {
		// Use the specified branch
		targetBranch = &git.Branch{Name: opt.Branch, ShortName: opt.Branch}
	} else {
		// Use current branch
		targetBranch, err = repo.CurrentBranch(ctx)
		if err != nil {
			return err
		}
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

	// Switch back to the original branch (current branch, not target branch)
	originalBranch, err := repo.CurrentBranch(ctx)
	if err != nil {
		return err
	}
	if err := repo.Checkout(ctx, originalBranch); err != nil {
		return err
	}

	return nil
}
