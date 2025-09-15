package workspaces

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/justinsb/gitflow/pkg/git"
)

func AddCommand(ctx context.Context, parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces",
	}
	var opt Options
	opt.InitDefaults()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return Run(cmd.Context(), opt, args)
	}
	parent.AddCommand(cmd)
}

type Options struct {
}

func (o *Options) InitDefaults() {

}

func Run(ctx context.Context, opt Options, args []string) error {
	repo, err := git.OpenRepo(ctx)
	if err != nil {
		return err
	}
	defer repo.Close()

	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	workspaceName := args[0]

	githubUser := ""
	githubBranch := ""

	if strings.Contains(workspaceName, ":") {
		tokens := strings.SplitN(workspaceName, ":", 2)
		githubUser = tokens[0]
		githubBranch = tokens[1]
	}
	if githubUser == "" || githubBranch == "" {
		return fmt.Errorf("workspace name must be in the format 'user:branch'")
	}

	branch := &git.Branch{
		Name: githubBranch,
	}
	if err := repo.Checkout(ctx, branch); err != nil {
		return err
	}

	return nil
}
