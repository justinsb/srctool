package git

import "context"

type Branch struct {
	Name      string
	ShortName string
	Remote    *Remote
}

func (b *Branch) String() string {
	return b.Name
}

func (r *Repo) DeleteBranch(ctx context.Context, branchName string) error {
	result, err := r.ExecGit(ctx, "branch", "-D", branchName)
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}
		return err
	}
	return nil
}
