package git

import "context"

type Branch struct {
	Name      string
	ShortName string
	Remote    *Remote
}

func (r *Repo) DeleteBranch(ctx context.Context, branchName string) error {
	_, err := r.ExecGit(ctx, "branch", "-D", branchName)
	return err
}
