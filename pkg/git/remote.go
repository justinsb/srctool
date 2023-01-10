package git

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

type Remote struct {
	repo     *Repo
	Name     string
	FetchURL string
	PushURL  string
}

func (r *Remote) ListBranches(ctx context.Context) ([]*Branch, error) {
	// log := klog.FromContext(ctx)

	repo := r.repo
	result, err := repo.ExecGit(ctx, "show-ref")
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return nil, err
	}

	remotePrefix := "refs/remotes/" + r.Name + "/"
	var branches []*Branch
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("error parsing output: %w", err)
			}
			break
		}

		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("unexpected line %q (expected 2 tokens)", line)
		}

		// sha := tokens[0]
		refName := tokens[1]

		if !strings.HasPrefix(refName, remotePrefix) {
			continue
		}
		shortName := strings.TrimPrefix(refName, remotePrefix)

		branches = append(branches, &Branch{
			Name:      r.Name + "/" + shortName,
			ShortName: shortName,
			Remote:    r,
		})
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

func (r *Remote) Rename(ctx context.Context, newName string) error {
	log := klog.FromContext(ctx)
	log.Info("renaming remote", "oldName", r.Name, "newName", newName)
	repo := r.repo
	result, err := repo.ExecGit(ctx, "remote", "rename", r.Name, newName)
	if err != nil {
		if result.ExitCode != 0 {
			result.PrintOutput()
		}

		return err
	}
	r.Name = newName
	return nil
}

func (r *Remote) UpdateURLs(ctx context.Context, fetchURL string, pushURL string) error {
	log := klog.FromContext(ctx)

	if r.FetchURL != fetchURL {
		log.Info("setting url", "remote", r.Name, "url", fetchURL)
		result, err := r.repo.ExecGit(ctx, "remote", "set-url", r.Name, fetchURL)
		if err != nil {
			if result.ExitCode != 0 {
				result.PrintOutput()
			}

			return err
		}

		r.FetchURL = fetchURL
		r.PushURL = fetchURL
	}

	if r.PushURL != pushURL {
		log.Info("setting push url", "remote", r.Name, "url", pushURL)
		result, err := r.repo.ExecGit(ctx, "remote", "set-url", "--push", r.Name, pushURL)
		if err != nil {
			if result.ExitCode != 0 {
				result.PrintOutput()
			}

			return err
		}
	}

	return nil
}
