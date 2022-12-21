package git

import (
	"context"
	"fmt"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

type Remote struct {
	repo *Repo
	Name string
	URL  string
}

func (r *Remote) ListBranches(ctx context.Context) ([]*Branch, error) {
	remotePrefix := r.Name + "/"

	it, err := r.repo.gitRepo.NewBranchIterator(git.BranchRemote)
	if err != nil {
		return nil, fmt.Errorf("error iterating branches: %w", err)
	}
	defer it.Free()

	var branches []*Branch
	if err := it.ForEach(func(branch *git.Branch, branchType git.BranchType) error {
		name, err := branch.Name()
		if err != nil {
			return fmt.Errorf("error getting branch name: %w", err)
		}
		if !strings.HasPrefix(name, remotePrefix) {
			return nil
		}
		shortName := strings.TrimPrefix(name, remotePrefix)

		branches = append(branches, &Branch{
			Name:      name,
			ShortName: shortName,
			Remote:    r,
		})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error iterating branches: %w", err)
	}

	return branches, nil
}

func (r *Remote) Fetch(ctx context.Context) error {
	repo := r.repo
	result, err := repo.ExecGit(ctx, "fetch", r.Name)
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return err
	}
	return nil
}
