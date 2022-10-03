package toc

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use: "toc",
	}
	var opt Options
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	parent.AddCommand(cmd)
}

type Options struct {
}

func Run(ctx context.Context, opt Options) error {
	repo, err := git.OpenRepo(ctx)
	if err != nil {
		return err
	}
	defer repo.Close()

	upstream, err := repo.FindUpstreamBranch(ctx)
	if err != nil {
		return err
	}

	if _, err := repo.ExecGitInteractive(ctx, "log", "--oneline", upstream.Name+"...", "--reverse"); err != nil {
		return err
	}

	return nil
}
