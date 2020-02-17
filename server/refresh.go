package server

import (
	"context"

	"github.com/google/go-github/v29/github"
	"github.com/pkg/errors"
	"github.com/ridge/bulldozer/pull"
	"github.com/ridge/bulldozer/server/handler"
	"github.com/ridge/go-githubapp/githubapp"
	"github.com/rs/zerolog"
)

func listRepos(ctx context.Context, client *github.Client) ([]*github.Repository, error) {
	opt := github.ListOptions{PerPage: 100}

	repositories := []*github.Repository{}
	for {
		pageRepositories, res, err := client.Apps.ListRepos(ctx, &opt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list repositories")
		}
		repositories = append(repositories, pageRepositories...)
		if res.NextPage == 0 {
			break
		}
		opt.Page = res.NextPage
	}
	return repositories, nil
}

func refreshRepo(ctx context.Context, serverConfig *handler.ServerConfig, repo *github.Repository, client *github.Client, logger zerolog.Logger) {
	prs, err := pull.ListOpenPullRequests(ctx, client, repo.GetOwner().GetLogin(), repo.GetName())
	if err != nil {
		logger.Warn().Err(errors.WithStack(err)).Msgf("Error enumerating all PRs in repository %s", repo.GetFullName())
		return
	}

	for _, pr := range prs {
		logger.Debug().Msgf("Handling %s#%d", repo.GetFullName(), pr.GetNumber())
		pullCtx := pull.NewGithubContext(client, pr)

		if err := handler.ProcessPullRequest(ctx, serverConfig, pullCtx, client, pr.GetBase().GetRef()); err != nil {
			logger.Warn().Err(errors.WithStack(err)).Msgf("Error processing PR %d in repository %s", pr.GetNumber(), repo.GetFullName())
		}
		logger.Debug().Msgf("Finished handling %s#%d", repo.GetFullName(), pr.GetNumber())
	}
}

func refresh(serverConfig *handler.ServerConfig, clientCreator githubapp.ClientCreator, logger zerolog.Logger) {
	logger.Info().Msgf("Refreshing existing PRs")
	appClient, err := clientCreator.NewAppClient()
	if err != nil {
		logger.Warn().Err(errors.WithStack(err)).Msg("Error creating GH app client in refresh")
		return
	}

	instService := githubapp.NewInstallationsService(appClient)

	installations, err := instService.ListAll(logger.WithContext(context.Background()))
	if err != nil {
		logger.Warn().Err(errors.WithStack(err)).Msg("Error enumerating all installations in refresh")
		return
	}

	for _, installation := range installations {
		logger.Debug().Msgf("Handling installation %d", installation.ID)

		ic, err := clientCreator.NewInstallationClient(installation.ID)
		if err != nil {
			logger.Warn().Err(errors.WithStack(err)).Msgf("Error creating GitHub client for installation %d in refresh, skipping it", installation.ID)
			continue
		}

		repos, err := listRepos(logger.WithContext(context.Background()), ic)

		for _, repo := range repos {
			logger.Debug().Msgf("Handling repository %s of installation %d", repo.GetFullName(), installation.ID)
			refreshRepo(logger.WithContext(context.Background()), serverConfig, repo, ic, logger)
			logger.Debug().Msgf("Finished handling repository %s of installation %d", repo.GetFullName(), installation.ID)
		}

		logger.Debug().Msgf("Finished handling installation %d", installation.ID)
	}
	logger.Info().Msgf("Finished refreshing existing PRs")
}
