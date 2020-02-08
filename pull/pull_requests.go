// Copyright 2018 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pull

import (
	"context"
	"fmt"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type pullRequestsService interface {
	List(ctx context.Context, owner, repoName string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
}

// ListOpenPullRequestsForSHA returns all pull requests where the HEAD of the source branch
// in the pull request matches the given SHA.
func ListOpenPullRequestsForSHA(ctx context.Context, client *github.Client, owner, repoName, SHA string) ([]*github.PullRequest, error) {
	return listOpenPullRequestsForSHA(ctx, client.PullRequests, owner, repoName, SHA)
}

func listOpenPullRequestsForSHA(ctx context.Context, pullRequests pullRequestsService, owner, repoName, SHA string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest

	openPRs, err := listOpenPullRequests(ctx, pullRequests, owner, repoName)

	if err != nil {
		return nil, err
	}

	for _, openPR := range openPRs {
		if openPR.Head.GetSHA() == SHA {
			results = append(results, openPR)
		}
	}

	return results, nil
}

func ListOpenPullRequestsForRef(ctx context.Context, client *github.Client, owner, repoName, ref string) ([]*github.PullRequest, error) {
	return listOpenPullRequestsForRef(ctx, client.PullRequests, owner, repoName, ref)
}

func listOpenPullRequestsForRef(ctx context.Context, pullRequests pullRequestsService, owner, repoName, ref string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest
	logger := zerolog.Ctx(ctx)

	openPRs, err := listOpenPullRequests(ctx, pullRequests, owner, repoName)

	if err != nil {
		return nil, err
	}

	for _, openPR := range openPRs {
		if fmt.Sprintf("refs/heads/%s", openPR.GetBase().GetRef()) == ref {
			results = append(results, openPR)
		}
	}

	if len(openPRs) > 0 {
		msg := "found open pull requests:"
		for _, openPR := range openPRs {
			msg += fmt.Sprintf(" %d (%s)", openPR.GetNumber(), fmt.Sprintf("refs/heads/%s", openPR.GetBase().GetRef()))
		}
		logger.Debug().Msg(msg)
	} else {
		logger.Debug().Msg("found no open pull requests")
	}

	return results, nil
}

func ListOpenPullRequests(ctx context.Context, client *github.Client, owner, repoName string) ([]*github.PullRequest, error) {
	return listOpenPullRequests(ctx, client.PullRequests, owner, repoName)
}

func listOpenPullRequests(ctx context.Context, pullRequests pullRequestsService, owner, repoName string) ([]*github.PullRequest, error) {
	var results []*github.PullRequest

	opts := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := pullRequests.List(ctx, owner, repoName, opts)
		if err != nil {
			return results, errors.Wrapf(err, "failed to list pull requests for repository %s/%s", owner, repoName)
		}
		for _, pr := range prs {
			results = append(results, pr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return results, nil
}
