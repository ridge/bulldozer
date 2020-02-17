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

package handler

import (
	"context"
	"encoding/json"

	"github.com/google/go-github/v29/github"
	"github.com/pkg/errors"
	"github.com/ridge/go-githubapp/githubapp"

	"github.com/ridge/bulldozer/pull"
)

type IssueComment struct {
	Config *ServerConfig
}

func (h *IssueComment) Handles() []string {
	return []string{"issue_comment"}
}

func handleIssueComment(config *ServerConfig, event github.IssueCommentEvent) {
	ctx := context.Background()

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	number := event.GetIssue().GetNumber()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, number)

	client, err := config.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to instantiate github client")
		return
	}

	pr, _, err := client.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), number)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to get pull request %s/%s#%d", owner, repoName, number)
		return
	}
	pullCtx := pull.NewGithubContext(client, pr)

	if err := ProcessPullRequest(ctx, config, pullCtx, client, pr.GetBase().GetRef()); err != nil {
		logger.Error().Err(err).Msg("Error processing pull request")
	}
}

func (h *IssueComment) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.IssueCommentEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment payload")
	}

	go handleIssueComment(h.Config, event)

	return nil

}

// type assertion
var _ githubapp.EventHandler = &IssueComment{}
