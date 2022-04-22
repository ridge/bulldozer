// Copyright 2020 Tectonic Labs, Inc.
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
	"errors"
	"testing"

	"github.com/google/go-github/v43/github"
	"github.com/stretchr/testify/require"
)

func int64Ptr(i int64) *int64 {
	return &i
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

type mockPullRequestsService struct{}

func (mockPullRequestsService) List(ctx context.Context, owner, repoName string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	if owner != "test-user" || repoName != "test-repo" {
		return nil, nil, errors.New("don't know this repo")
	}
	if opts.State != "open" {
		return nil, nil, errors.New("don't know how to enumerate closed PRs")
	}
	switch opts.ListOptions.Page {
	case 0, 1:
		return []*github.PullRequest{
				{
					ID:     int64Ptr(100004),
					Number: intPtr(4),
					State:  stringPtr("open"),
					Base: &github.PullRequestBranch{
						Ref: stringPtr("devel"),
					},
					Head: &github.PullRequestBranch{
						SHA: stringPtr("00facade"),
					},
				},
				{
					ID:     int64Ptr(100003),
					Number: intPtr(3),
					State:  stringPtr("open"),
					Base: &github.PullRequestBranch{
						Ref: stringPtr("master"),
					},
					Head: &github.PullRequestBranch{
						SHA: stringPtr("deadbeef"),
					},
				},
			}, &github.Response{
				NextPage: 2,
				LastPage: 2,
			}, nil
	case 2:
		return []*github.PullRequest{
				{
					ID:     int64Ptr(100002),
					Number: intPtr(2),
					State:  stringPtr("open"),
					Base: &github.PullRequestBranch{
						Ref: stringPtr("master"),
					},
					Head: &github.PullRequestBranch{
						SHA: stringPtr("cafebabe"),
					},
				},
				{
					ID:     int64Ptr(100001),
					Number: intPtr(1),
					State:  stringPtr("open"),
					Base: &github.PullRequestBranch{
						Ref: stringPtr("devel"),
					},
					Head: &github.PullRequestBranch{
						SHA: stringPtr("feedface"),
					},
				},
			}, &github.Response{
				PrevPage:  1,
				FirstPage: 1,
			}, nil
	default:
		return nil, nil, errors.New("don't know about other pages")
	}
}

func numbers(prs []*github.PullRequest) []int {
	out := []int{}
	for _, pr := range prs {
		out = append(out, *pr.Number)
	}
	return out
}

func TestMissingRepo(t *testing.T) {
	_, err := listOpenPullRequests(context.Background(), mockPullRequestsService{}, "test-user", "no-such-repo")
	require.Error(t, err)
}

func TestListOpenPullRequests(t *testing.T) {
	prs, err := listOpenPullRequests(context.Background(), mockPullRequestsService{}, "test-user", "test-repo")
	require.NoError(t, err)

	require.ElementsMatch(t, []int{1, 2, 3, 4}, numbers(prs))
}

func TestListOpenPullRequestsForRef(t *testing.T) {
	prs, err := listOpenPullRequestsForRef(context.Background(), mockPullRequestsService{}, "test-user", "test-repo", "refs/heads/master")
	require.NoError(t, err)

	require.ElementsMatch(t, []int{2, 3}, numbers(prs))
}

func TestMissingRepoForRef(t *testing.T) {
	_, err := listOpenPullRequestsForRef(context.Background(), mockPullRequestsService{}, "test-user", "no-such-repo", "refs/heads/master")
	require.Error(t, err)
}

func TestListOpenPullRequestsForSHA(t *testing.T) {
	prs, err := listOpenPullRequestsForSHA(context.Background(), mockPullRequestsService{}, "test-user", "test-repo", "deadbeef")
	require.NoError(t, err)

	require.ElementsMatch(t, []int{3}, numbers(prs))
}

func TestMissingRepoForSHA(t *testing.T) {
	_, err := listOpenPullRequestsForSHA(context.Background(), mockPullRequestsService{}, "test-user", "no-such-repo", "cafebabe")
	require.Error(t, err)
}
