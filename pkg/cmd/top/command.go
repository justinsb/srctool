package top

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use: "top",
	}
	var opt Options
	opt.InitDefaults()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	parent.AddCommand(cmd)
}

type Options struct {
	N int
}

func (o *Options) InitDefaults() {
	o.N = 10
}

func Run(ctx context.Context, opt Options) error {
	repo, err := git.OpenRepo(ctx)
	if err != nil {
		return err
	}
	defer repo.Close()

	args := []string{
		"for-each-ref", "--sort=-committerdate", "--count=" + strconv.Itoa(opt.N), "--format=%(refname:short)", "refs/heads",
	}
	if _, err := repo.ExecGitInteractive(ctx, args...); err != nil {
		return err
	}

	return nil
}
