package rebase

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use: "rebase",
	}
	var opt Options
	opt.InitDefaults()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt)
	}
	cmd.Flags().BoolVarP(&opt.Interactive, "interactive", "i", opt.Interactive, "run rebase interactively")
	parent.AddCommand(cmd)
}

type Options struct {
	Interactive bool
	Verbose     bool
}

func (o *Options) InitDefaults() {

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

	if err := upstream.Remote.Fetch(ctx); err != nil {
		return err
	}

	args := []string{"rebase"}
	if opt.Interactive {
		args = append(args, "-i")
	}
	args = append(args, "--autosquash")
	args = append(args, upstream.Name)

	if _, err := repo.ExecGitInteractive(ctx, args...); err != nil {
		return err
	}

	return nil
}
