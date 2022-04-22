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

	"github.com/google/go-github/v43/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/ridge/bulldozer/bulldozer"
	"github.com/ridge/bulldozer/pull"
)

type ServerConfig struct {
	githubapp.ClientCreator
	bulldozer.ConfigFetcher

	PushRestrictionUserToken string
}

func FindPRConfig(ctx context.Context, configFetcher bulldozer.ConfigFetcher, client *github.Client, pullCtx pull.Context) (*bulldozer.FetchedConfig, error) {
	logger := zerolog.Ctx(ctx)

	config, err := configFetcher.ConfigForPR(ctx, client, pullCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch configuration")
	}

	switch {
	case config.Missing():
		logger.Debug().Msgf("No configuration found for %s", config)
		return nil, nil
	case config.Invalid():
		logger.Warn().Msgf("Configuration is invalid for %s %+v", config, config.Error)
		return nil, nil
	default:
		return &config, nil
	}
}

func UpdatePullRequest(ctx context.Context, serverConfig *ServerConfig, prConfig bulldozer.Config, pullCtx pull.Context, client *github.Client, baseRef string) error {
	shouldUpdate, err := bulldozer.ShouldUpdatePR(ctx, pullCtx, prConfig.Update)
	if err != nil {
		return errors.Wrap(err, "unable to determine update status")
	}

	if !shouldUpdate {
		return nil
	}

	if err := bulldozer.UpdatePR(ctx, pullCtx, client, prConfig.Update, baseRef); err != nil {
		return errors.Wrap(err, "failed to update pull request")
	}

	return nil
}

func MergePullRequest(ctx context.Context, serverConfig *ServerConfig, prConfig bulldozer.Config, pullCtx pull.Context, client *github.Client) error {
	shouldMerge, err := bulldozer.ShouldMergePR(ctx, pullCtx, prConfig.Merge)
	if err != nil {
		return errors.Wrap(err, "unable to determine merge status")
	}
	if !shouldMerge {
		return nil
	}

	merger := bulldozer.NewGitHubMerger(client)
	if serverConfig.PushRestrictionUserToken != "" {
		tokenClient, err := serverConfig.NewTokenClient(serverConfig.PushRestrictionUserToken)
		if err != nil {
			return errors.Wrap(err, "failed to create token client")
		}
		merger = bulldozer.NewPushRestrictionMerger(merger, bulldozer.NewGitHubMerger(tokenClient))
	}

	if err := bulldozer.MergePR(ctx, pullCtx, merger, prConfig.Merge); err != nil {
		return errors.Wrap(err, "failed to merge pull request")
	}

	return nil
}

func ProcessPullRequest(ctx context.Context, serverConfig *ServerConfig, pullCtx pull.Context, client *github.Client, baseRef string) error {
	logger := zerolog.Ctx(ctx)

	bulldozerConfig, err := FindPRConfig(ctx, serverConfig.ConfigFetcher, client, pullCtx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch configuration")
	}
	if bulldozerConfig == nil {
		return nil
	}

	logger.Debug().Msgf("Found valid configuration for %s", bulldozerConfig)
	prConfig := *bulldozerConfig.Config

	if err := UpdatePullRequest(ctx, serverConfig, prConfig, pullCtx, client, baseRef); err != nil {
		logger.Error().Err(err).Msg("Update failed")
	}

	return MergePullRequest(ctx, serverConfig, prConfig, pullCtx, client)
}
