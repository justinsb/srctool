package top

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "shows the most recently changed branches",
	}
	var opt Options
	opt.InitDefaults()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				cmd.Usage()
				cmd.PrintErrln()
				return fmt.Errorf("expected [N]")
			}
			opt.N = n
		}
		return nil
	}
	cmd.Flags().IntVar(&opt.N, "n", opt.N, "max number of branches to show")
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
