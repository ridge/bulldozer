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

	"github.com/google/go-github/v43/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"

	"github.com/ridge/bulldozer/pull"
)

type Status struct {
	Config *ServerConfig
}

func (h *Status) Handles() []string {
	return []string{"status"}
}

func handleStatus(config *ServerConfig, event github.StatusEvent) {
	ctx := context.Background()

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PrepareRepoContext(ctx, installationID, repo)

	if event.GetState() != "success" {
		logger.Debug().Msgf("Doing nothing since context state for %q was %q", event.GetContext(), event.GetState())
		return
	}

	client, err := config.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to instantiate github client")
		return
	}

	prs, err := pull.ListOpenPullRequestsForSHA(ctx, client, owner, repoName, event.GetSHA())
	if err != nil {
		logger.Error().Err(err).Msg("failed to determine open pull requests matching the status context change")
		return
	}

	if len(prs) == 0 {
		logger.Debug().Msg("Doing nothing since status change event affects no open pull requests")
		return
	}

	for _, pr := range prs {
		pullCtx := pull.NewGithubContext(client, pr)
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()

		if err := ProcessPullRequest(ctx, config, pullCtx, client, pr.GetBase().GetRef()); err != nil {
			logger.Error().Err(err).Msg("Error updating pull request")
		}
	}

}

func (h *Status) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.StatusEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse status event payload")
	}

	go handleStatus(h.Config, event)

	return nil
}

// type assertion
var _ githubapp.EventHandler = &Status{}
