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

	"github.com/google/go-github/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/palantir/bulldozer/bulldozer"
	"github.com/palantir/bulldozer/pull"
)

type Base struct {
	githubapp.ClientCreator
	bulldozer.ConfigFetcher

	PushRestrictionUserToken string
}

func FindConfig(ctx context.Context, configFetcher bulldozer.ConfigFetcher, client *github.Client, pullCtx pull.Context) (*bulldozer.FetchedConfig, error) {
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

func (b *Base) ProcessPullRequest(ctx context.Context, pullCtx pull.Context, client *github.Client) error {
	logger := zerolog.Ctx(ctx)

	bulldozerConfig, err := FindConfig(ctx, b.ConfigFetcher, client, pullCtx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch configuration")
	}
	if bulldozerConfig == nil {
		return nil
	}

	logger.Debug().Msgf("Found valid configuration for %s", bulldozerConfig)
	config := *bulldozerConfig.Config

	shouldMerge, err := bulldozer.ShouldMergePR(ctx, pullCtx, config.Merge)
	if err != nil {
		return errors.Wrap(err, "unable to determine merge status")
	}
	if !shouldMerge {
		return nil
	}

	merger := bulldozer.NewGitHubMerger(client)
	if b.PushRestrictionUserToken != "" {
		tokenClient, err := b.NewTokenClient(b.PushRestrictionUserToken)
		if err != nil {
			return errors.Wrap(err, "failed to create token client")
		}
		merger = bulldozer.NewPushRestrictionMerger(merger, bulldozer.NewGitHubMerger(tokenClient))
	}

	if err := bulldozer.MergePR(ctx, pullCtx, merger, config.Merge); err != nil {
		return errors.Wrap(err, "failed to merge pull request")
	}

	return nil
}

func (b *Base) UpdatePullRequest(ctx context.Context, pullCtx pull.Context, client *github.Client, baseRef string) error {
	logger := zerolog.Ctx(ctx)

	bulldozerConfig, err := FindConfig(ctx, b.ConfigFetcher, client, pullCtx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch configuration")
	}
	if bulldozerConfig == nil {
		return nil
	}

	logger.Debug().Msgf("Found valid configuration for %s", bulldozerConfig)
	config := *bulldozerConfig.Config

	shouldUpdate, err := bulldozer.ShouldUpdatePR(ctx, pullCtx, config.Update)
	if err != nil {
		return errors.Wrap(err, "unable to determine update status")
	}

	if !shouldUpdate {
		return nil
	}

	if err := bulldozer.UpdatePR(ctx, pullCtx, client, config.Update, baseRef); err != nil {
		return errors.Wrap(err, "failed to update pull request")
	}

	return nil
}
