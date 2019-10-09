package github

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"gopkg.in/yaml.v2"
)

// Client used to call the github api
type Client struct {
	github *github.Client
	token  string
}

// Settings contains the settings to be apply to a github repository
type Settings struct {
	Repository repository
	Labels     []label
	Branches   []branch
	Webhooks   []webhook
	Topics     []string
}

type repository struct {
	Name             string
	Owner            string
	Description      string
	Homepage         string
	DefaultBranch    string
	Private          bool
	HasIssues        bool
	HasProjects      bool
	HasPages         bool
	HasWiki          bool
	HasDownloads     bool
	IsTemplate       bool
	Archived         bool
	AllowSquashMerge bool
	AllowMergeCommit bool
	AllowRebaseMerge bool
}

type label struct {
	Name        string
	Description string
	Color       string
}

type branch struct {
	Name       string
	Protection protection
}

type protection struct {
	Enabled                      bool
	EnforceAdmins                bool
	RequiredApprovingReviewCount requiredApprovingReviewCount
	RequiredStatusChecks         requiredStatusChecks
}

type requiredApprovingReviewCount struct {
	RequiredApprovingReviewCount int
	DismissStaleReviews          bool
	RequireCodeOwnerReviews      bool
}

type requiredStatusChecks struct {
	Strict   bool
	Contexts []string
}

type webhook struct {
	ID          int64
	URL         string
	ContentType string
	Secret      string
	Events      []string
}

// New creates a new client
func New(token string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(context.Background(), ts)

	return &Client{
		github: github.NewClient(tc),
		token:  token,
	}
}

// GetSettingsFromFile parse a yaml file containing settings
func GetSettingsFromFile(file string) (*Settings, error) {
	content, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, errors.Wrap(err, "Error while reading settings file")
	}

	var settings Settings
	err = yaml.Unmarshal(content, &settings)

	if err != nil {
		return nil, errors.Wrap(err, "Error while unmarshal settings")
	}

	for i, branch := range settings.Branches {
		settings.Branches[i].Protection.Enabled = true
		if branch.Protection.RequiredApprovingReviewCount.RequiredApprovingReviewCount == 0 {
			settings.Branches[i].Protection.RequiredApprovingReviewCount.DismissStaleReviews = false
			settings.Branches[i].Protection.RequiredApprovingReviewCount.RequireCodeOwnerReviews = false
		}
	}

	return &settings, nil
}

// Apply the specified settings to a repository
func (client *Client) Apply(settings *Settings) error {
	githubSettings, err := client.GetSettingsFromGithub(settings.Repository.Owner, settings.Repository.Name)

	if err != nil {
		return errors.Wrap(err, "Error getting settings from github")
	}

	err = client.updateRepoSettings(settings.Repository.Owner, settings.Repository.Name, githubSettings.Repository, settings.Repository)

	if err != nil {
		return errors.Wrap(err, "Error updating repository settings")
	}

	err = client.updateLabels(settings.Repository.Owner, settings.Repository.Name, githubSettings.Labels, settings.Labels)

	if err != nil {
		return errors.Wrap(err, "Error updating repository labels")
	}

	err = client.updateBranchSettings(settings.Repository.Owner, settings.Repository.Name, githubSettings.Branches, settings.Branches)

	if err != nil {
		return errors.Wrap(err, "Error updating repository branches protection")
	}

	err = client.updateWebhooks(settings.Repository.Owner, settings.Repository.Name, githubSettings.Webhooks, settings.Webhooks)

	if err != nil {
		return errors.Wrap(err, "Error updating repository webhooks")
	}

	err = client.updateTopicsSettings(settings.Repository.Owner, settings.Repository.Name, githubSettings.Topics, settings.Topics)

	if err != nil {
		return errors.Wrap(err, "Error updating repository topics")
	}

	return nil
}

// GetSettingsFromGithub returns the settings current applied on a github repository
func (client *Client) GetSettingsFromGithub(owner string, name string) (*Settings, error) {
	githubRepo, _, err := client.github.Repositories.Get(context.Background(), owner, name)

	if err != nil {
		return nil, errors.Wrap(err, "Error while getting repository from github")
	}

	githubLabels, _, err := client.github.Issues.ListLabels(context.Background(), owner, name, &github.ListOptions{})

	if err != nil {
		return nil, errors.Wrap(err, "Error while getting labels from github")
	}

	labelSettings := make([]label, 0, len(githubLabels))

	for _, githubLabel := range githubLabels {
		labelSettings = append(labelSettings, label{
			Name:        githubLabel.GetName(),
			Description: githubLabel.GetDescription(),
			Color:       githubLabel.GetColor(),
		})
	}

	branchesSettings := []branch{}

	githubBranches, _, err := client.github.Repositories.ListBranches(context.Background(), owner, name, &github.ListOptions{})

	if err != nil {
		return nil, errors.Wrap(err, "Error while listing branches")
	}

	for _, githubBranch := range githubBranches {
		if githubBranch.GetProtected() {
			githubProtection, _, err := client.github.Repositories.GetBranchProtection(context.Background(), owner, name, githubBranch.GetName())

			if err != nil {
				return nil, errors.Wrap(err, "Error while getting branch protection")
			}

			var requiredReview requiredApprovingReviewCount

			if githubProtection.RequiredPullRequestReviews != nil {
				requiredReview = requiredApprovingReviewCount{
					RequiredApprovingReviewCount: githubProtection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
					RequireCodeOwnerReviews:      githubProtection.RequiredPullRequestReviews.RequireCodeOwnerReviews,
					DismissStaleReviews:          githubProtection.RequiredPullRequestReviews.DismissStaleReviews,
				}
			}

			branchesSettings = append(branchesSettings, branch{
				Name: githubBranch.GetName(),
				Protection: protection{
					Enabled:                      true,
					EnforceAdmins:                githubProtection.GetEnforceAdmins().Enabled,
					RequiredApprovingReviewCount: requiredReview,
					RequiredStatusChecks: requiredStatusChecks{
						Strict:   githubProtection.RequiredStatusChecks.Strict,
						Contexts: githubProtection.RequiredStatusChecks.Contexts,
					},
				},
			})
		} else {
			branchesSettings = append(branchesSettings, branch{Name: githubBranch.GetName()})
		}
	}

	hooks, _, err := client.github.Repositories.ListHooks(context.Background(), owner, name, &github.ListOptions{})

	if err != nil {
		return nil, errors.Wrap(err, "Error getting webhooks")
	}

	webhooksSettings := make([]webhook, 0, len(hooks))

	for _, hook := range hooks {
		webhooksSettings = append(webhooksSettings, webhook{
			ID:          hook.GetID(),
			URL:         hook.Config["url"].(string),
			ContentType: hook.Config["content_type"].(string),
			Secret:      hook.Config["secret"].(string),
			Events:      hook.Events,
		})
	}

	return &Settings{
		Topics: githubRepo.Topics,
		Repository: repository{
			Name:             githubRepo.GetName(),
			Owner:            githubRepo.Owner.GetLogin(),
			Description:      githubRepo.GetDescription(),
			Homepage:         githubRepo.GetHomepage(),
			DefaultBranch:    githubRepo.GetDefaultBranch(),
			Private:          githubRepo.GetPrivate(),
			HasIssues:        githubRepo.GetHasIssues(),
			HasProjects:      githubRepo.GetHasProjects(),
			HasWiki:          githubRepo.GetHasWiki(),
			HasDownloads:     githubRepo.GetHasDownloads(),
			IsTemplate:       githubRepo.GetIsTemplate(),
			AllowSquashMerge: githubRepo.GetAllowSquashMerge(),
			AllowMergeCommit: githubRepo.GetAllowMergeCommit(),
			AllowRebaseMerge: githubRepo.GetAllowRebaseMerge(),
			Archived:         githubRepo.GetArchived(),
		},
		Labels:   labelSettings,
		Branches: branchesSettings,
		Webhooks: webhooksSettings,
	}, nil
}

func (client *Client) createBranch(branches []string, url string) error {
	repo, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL: url,
	})

	if err != nil {
		return errors.Wrap(err, "Error initializing git repository")
	}

	headRef, err := repo.Head()

	if err != nil {
		return errors.Wrap(err, "Error getting repository head")
	}

	for _, branch := range branches {
		err = repo.Storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/"+branch), headRef.Hash()))

		if err != nil {
			return errors.Wrapf(err, "Error setting reference storer for branch %s", branch)
		}

		err = repo.Push(&git.PushOptions{})

		if err != nil {
			return errors.Wrapf(err, "Error pushing reference for branch %s", branch)
		}
	}

	return nil
}

func fmtGithubURL(owner, name, token string) string {
	return fmt.Sprintf("https://%s@github.com/%s/%s.git", token, owner, name)
}
