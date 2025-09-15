package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/justinsb/gitflow/pkg/cmd/cherry"
	"github.com/justinsb/gitflow/pkg/cmd/forks"
	"github.com/justinsb/gitflow/pkg/cmd/pr"
	"github.com/justinsb/gitflow/pkg/cmd/prune"
	"github.com/justinsb/gitflow/pkg/cmd/rebase"
	"github.com/justinsb/gitflow/pkg/cmd/toc"
	"github.com/justinsb/gitflow/pkg/cmd/top"
	"github.com/justinsb/gitflow/pkg/cmd/workspaces"
)

func main() {
	err := Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Run(ctx context.Context) error {
	root := &cobra.Command{
		Use: "gitflow",
	}
	root.SilenceErrors = true
	root.SilenceUsage = true

	var goflags flag.FlagSet
	klog.InitFlags(&goflags)
	root.PersistentFlags().AddGoFlag(goflags.Lookup("v"))

	rebase.AddCommand(ctx, root)
	toc.AddCommand(ctx, root)
	prune.AddCommand(ctx, root)
	top.AddCommand(ctx, root)
	pr.AddCommand(ctx, root)
	forks.AddCommand(ctx, root)
	cherry.AddCommand(ctx, root)
	workspaces.AddCommand(ctx, root)

	return root.ExecuteContext(ctx)
}
