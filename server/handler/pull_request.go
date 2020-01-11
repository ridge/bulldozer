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

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"github.com/ridge/go-githubapp/githubapp"

	"github.com/ridge/bulldozer/pull"
)

type PullRequest struct {
	Config *ServerConfig
}

func (h *PullRequest) Handles() []string {
	return []string{"pull_request"}
}

func handlePullRequest(config *ServerConfig, event github.PullRequestEvent) {
	ctx := context.Background()

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	number := event.GetNumber()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, number)

	if event.GetAction() == "closed" {
		logger.Debug().Msg("Doing nothing since pull request is closed")
		return
	}

	client, err := config.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to instantiate github client")
		return
	}

	pr, _, err := client.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to get pull request %s/%s#%d", owner, repoName, number)
		return
	}
	pullCtx := pull.NewGithubContext(client, pr)

	if err := ProcessPullRequest(ctx, config, pullCtx, client, pr.GetBase().GetRef()); err != nil {
		logger.Error().Err(err).Msg("Error updating pull request")
	}

}

func (h *PullRequest) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse pull request event payload")
	}

	go handlePullRequest(h.Config, event)

	return nil
}

// type assertion
var _ githubapp.EventHandler = &PullRequest{}
