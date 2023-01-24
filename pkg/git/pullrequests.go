package git

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v49/github"
	"k8s.io/klog/v2"
)

type ForgeInfo interface {
}

type GithubForgeInfo struct {
	Organization string
	Repository   string
}

func ParseRepoFromURL(ctx context.Context, s string) ForgeInfo {
	if strings.HasPrefix(s, "https://github.com/") {
		s = strings.TrimSuffix(s, ".git")

		u, err := url.Parse(s)
		if err != nil {
			klog.Warningf("unable to parse url %q: %v", s, err)
			return nil
		}
		pathTokens := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(pathTokens) == 2 {
			return &GithubForgeInfo{
				Organization: pathTokens[0],
				Repository:   pathTokens[1],
			}
		}
	}

	if strings.HasPrefix(s, "git@github.com:") {
		s = strings.TrimPrefix(s, "git@github.com:")
		s = strings.TrimSuffix(s, ".git")

		tokens := strings.Split(strings.Trim(s, "/"), "/")
		if len(tokens) == 2 {
			return &GithubForgeInfo{
				Organization: tokens[0],
				Repository:   tokens[1],
			}
		}
	}

	klog.Warningf("unknown repo %q", s)
	return nil
}

func (r *Remote) GetPullRequest(ctx context.Context, id string) (*GithubPullRequest, error) {
	info := ParseRepoFromURL(ctx, r.FetchURL)
	if info == nil {
		return nil, fmt.Errorf("cannot determine forge from %q", r.FetchURL)
	}

	switch info := info.(type) {
	case *GithubForgeInfo:
		prNumber, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("invalid github pull request number %q", id)
		}

		client := github.NewClient(nil)
		pr, _, err := client.PullRequests.Get(ctx, info.Organization, info.Repository, prNumber)
		if err != nil {
			return nil, fmt.Errorf("error fetching pull request from github: %w", err)
		}
		commits, response, err := client.PullRequests.ListCommits(ctx, info.Organization, info.Repository, pr.GetNumber(), &github.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error fetching pull request commits from github: %w", err)
		}
		if response.NextPage != 0 || response.NextPageToken != "" {
			return nil, fmt.Errorf("commits response was paginated; too many commits")
		}

		return &GithubPullRequest{
			pr:      pr,
			commits: commits,
		}, nil
	default:
		return nil, fmt.Errorf("unknown forge type %T", info)
	}
}

type GithubPullRequest struct {
	pr      *github.PullRequest
	commits []*github.RepositoryCommit
}

func (r *GithubPullRequest) Commits() []string {
	var shas []string
	for _, commit := range r.commits {
		shas = append(shas, commit.GetSHA())
	}
	return shas
}

func (r *GithubPullRequest) Title() string {
	return r.pr.GetTitle()
}
