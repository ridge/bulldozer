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

package bulldozer

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/ridge/bulldozer/pull"
)

func statusDescriptionWhitelisted(description string, whitelist []string) bool {
	for _, rx := range whitelist {
		if regexp.MustCompile(rx).MatchString(description) {
			return true
		}
	}
	return false
}

func ShouldUpdatePR(ctx context.Context, pullCtx pull.Context, updateConfig UpdateConfig) (bool, error) {
	logger := zerolog.Ctx(ctx)

	if !updateConfig.DraftUpdate && pullCtx.IsDraft() {
		logger.Debug().Msgf("%s is deemed not updateable because it is in draft stage", pullCtx.Locator())
		return false, nil
	}

	if !updateConfig.Blacklist.Enabled() && !updateConfig.Whitelist.Enabled() && len(updateConfig.RequiredStatuses) == 0 {
		logger.Info().Msgf("%s is not updated due to missing update conditions in config file", pullCtx.Locator())
		return false, nil
	}

	if updateConfig.Blacklist.Enabled() {
		blacklisted, reason, err := IsPRBlacklisted(ctx, pullCtx, updateConfig.Blacklist)
		if err != nil {
			return false, errors.Wrapf(err, "failed to determine if pull request %s is blacklisted", pullCtx.Locator())
		}
		if blacklisted {
			logger.Debug().Msgf("%s is deemed not updateable because blacklisting is enabled and %s", pullCtx.Locator(), reason)
			return false, nil
		}
	}

	if updateConfig.Whitelist.Enabled() {
		whitelisted, reason, err := IsPRWhitelisted(ctx, pullCtx, updateConfig.Whitelist)
		if err != nil {
			return false, errors.Wrapf(err, "failed to determine if pull request %s is whitelisted", pullCtx.Locator())
		}
		if !whitelisted {
			logger.Debug().Msgf("%s is deemed not updateable because whitelisting is enabled and no whitelist signal detected", pullCtx.Locator())
			return false, nil
		}

		logger.Debug().Msgf("%s is whitelisted because whitelisting is enabled and %s", pullCtx.Locator(), reason)
	}

	if len(updateConfig.RequiredStatuses) > 0 {
		successStatuses, failedStatuses, err := pullCtx.CurrentStatuses(ctx)
		if err != nil {
			return false, errors.Wrapf(err, "failed to determine currently successful status checks for pull request %s", pullCtx.Locator())
		}

		for statusName, whitelist := range updateConfig.RequiredStatusesDescriptionWhitelist {
			statusDescription, hasStatus := failedStatuses[statusName]
			if !hasStatus {
				continue
			}

			if statusDescriptionWhitelisted(statusDescription, whitelist) {
				successStatuses = append(successStatuses, statusName)
			}
		}

		unsatisfiedStatuses := setDifference(updateConfig.RequiredStatuses, successStatuses)
		if len(unsatisfiedStatuses) > 0 {
			logger.Debug().Msgf("%s is deemed not updateable because of unfulfilled status checks: [%s]", pullCtx.Locator(),
				strings.Join(unsatisfiedStatuses, ","))
			return false, nil
		}
	}

	logger.Info().Msgf("%s is deemed updateable", pullCtx.Locator())

	return true, nil
}

func UpdatePR(ctx context.Context, pullCtx pull.Context, client *github.Client, updateConfig UpdateConfig, baseRef string) error {
	logger := zerolog.Ctx(ctx)

	//todo: should the updateConfig struct provide any other details here?

	go func(ctx context.Context, baseRef string) {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for i := 0; i < MaxPullRequestPollCount; i++ {
			<-ticker.C

			pr, _, err := client.PullRequests.Get(ctx, pullCtx.Owner(), pullCtx.Repo(), pullCtx.Number())
			if err != nil {
				logger.Error().Err(errors.WithStack(err)).Msgf("Failed to retrieve pull request %q", pullCtx.Locator())
				return
			}

			if pr.GetState() == "closed" {
				logger.Debug().Msg("Pull request already closed")
				return
			}

			if pr.Head.Repo.GetFork() {
				logger.Debug().Msg("Pull request is from a fork, cannot keep it up to date with base ref")
				return
			}

			comparison, _, err := client.Repositories.CompareCommits(ctx, pullCtx.Owner(), pullCtx.Repo(), baseRef, pr.GetHead().GetSHA(), nil)
			if err != nil {
				logger.Error().Err(errors.WithStack(err)).Msgf("cannot compare %s and %s for %q", baseRef, pr.GetHead().GetSHA(), pullCtx.Locator())
			}
			if comparison.GetBehindBy() > 0 {
				logger.Debug().Msg("Pull request is not up to date")

				mergeRequest := &github.RepositoryMergeRequest{
					Base: github.String(pr.Head.GetRef()),
					Head: github.String(baseRef),
				}

				mergeCommit, _, err := client.Repositories.Merge(ctx, pullCtx.Owner(), pullCtx.Repo(), mergeRequest)
				if err != nil {
					logger.Error().Err(errors.WithStack(err)).Msg("Merge failed unexpectedly")
				}

				logger.Info().Msgf("Successfully updated pull request from base ref %s as merge %s", baseRef, mergeCommit.GetSHA())
			} else {
				logger.Debug().Msg("Pull request is not out of date, not updating")
			}

			return
		}
	}(zerolog.Ctx(ctx).WithContext(context.Background()), baseRef)

	return nil
}
