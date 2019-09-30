package github

import (
	"context"
	"log"
	"sort"

	"reflect"

	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
)

func (client *Client) updateTopicsSettings(owner, name string, githubTopics, topics []string) error {
	sort.Strings(githubTopics)
	sort.Strings(topics)

	if reflect.DeepEqual(githubTopics, topics) {
		return nil
	}

	log.Print("[INFO] Updating repository topics\n")

	_, _, err := client.github.Repositories.ReplaceAllTopics(context.Background(), owner, name, topics)

	if err != nil {
		return errors.Wrap(err, "Error updating repository topics\n")
	}

	return nil
}

func (client *Client) updateRepoSettings(owner, name string, githubRepo, repo repository) error {
	if reflect.DeepEqual(githubRepo, repo) {
		return nil
	}

	log.Print("[INFO] Updating repository settings\n")

	_, _, err := client.github.Repositories.Edit(context.Background(), owner, name, &github.Repository{
		Description:      github.String(repo.Description),
		Homepage:         github.String(repo.Homepage),
		DefaultBranch:    github.String(repo.DefaultBranch),
		Private:          github.Bool(repo.Private),
		HasIssues:        github.Bool(repo.HasIssues),
		HasProjects:      github.Bool(repo.HasProjects),
		HasPages:         github.Bool(repo.HasPages),
		HasWiki:          github.Bool(repo.HasWiki),
		HasDownloads:     github.Bool(repo.HasDownloads),
		IsTemplate:       github.Bool(repo.IsTemplate),
		Archived:         github.Bool(repo.Archived),
		AllowSquashMerge: github.Bool(repo.AllowSquashMerge),
		AllowMergeCommit: github.Bool(repo.AllowMergeCommit),
		AllowRebaseMerge: github.Bool(repo.AllowRebaseMerge),
	})

	if err != nil {
		return errors.Wrap(err, "Error updating settings\n")
	}

	return nil
}

func (client *Client) updateLabels(owner, name string, githubLabels, labelsSettings []label) error {
	labelsToCreate := []label{}
	labelsToUpdate := []label{}
	deleteLabelMap := map[string]label{}

	for _, githubLabel := range githubLabels {
		deleteLabelMap[githubLabel.Name] = githubLabel
	}

	for _, labelSetting := range labelsSettings {
		githubLabel, ok := deleteLabelMap[labelSetting.Name]

		if !ok {
			labelsToCreate = append(labelsToCreate, labelSetting)
		} else {
			delete(deleteLabelMap, labelSetting.Name)

			if labelSetting != githubLabel {
				labelsToUpdate = append(labelsToUpdate, labelSetting)
			}
		}
	}

	for labelName := range deleteLabelMap {
		log.Printf("[INFO] Deleting label %s\n", labelName)

		_, err := client.github.Issues.DeleteLabel(context.Background(), owner, name, labelName)

		if err != nil {
			return errors.Wrap(err, "Error deleting a label\n")
		}
	}

	for _, newLabel := range labelsToCreate {
		log.Printf("[INFO] Creating label %s\n", newLabel.Name)

		_, _, err := client.github.Issues.CreateLabel(context.Background(), owner, name, &github.Label{
			Name:        github.String(newLabel.Name),
			Color:       github.String(newLabel.Color),
			Description: github.String(newLabel.Description),
		})

		if err != nil {
			return errors.Wrap(err, "Error creating a label\n")
		}
	}

	for _, updateLabel := range labelsToUpdate {
		log.Printf("[INFO] Updating label %s\n", updateLabel.Name)

		_, _, err := client.github.Issues.EditLabel(context.Background(), owner, name, updateLabel.Name, &github.Label{
			Name:        github.String(updateLabel.Name),
			Color:       github.String(updateLabel.Color),
			Description: github.String(updateLabel.Description),
		})

		if err != nil {
			return errors.Wrap(err, "Error updating a label\n")
		}
	}

	return nil
}

func (client *Client) updateBranchSettings(owner string, name string, githubBranches []branch, branchesSettings []branch) error {
	branchesToUpdate := []branch{}
	deleteBranchesMap := map[string]branch{}

	for _, githubBranch := range githubBranches {
		deleteBranchesMap[githubBranch.Name] = githubBranch
	}

	for _, branchSettings := range branchesSettings {
		githubBranch, ok := deleteBranchesMap[branchSettings.Name]

		if !ok {
			log.Printf("[INFO] Creating new branch %s\n", branchSettings.Name)

			err := client.createBranch(branchSettings.Name, fmtGithubURL(owner, name, client.token))

			if err != nil {
				return errors.Wrap(err, "Error creating branch\n")
			}

			branchesToUpdate = append(branchesToUpdate, branchSettings)
		} else {
			delete(deleteBranchesMap, branchSettings.Name)

			if !reflect.DeepEqual(githubBranch, branchSettings) {
				branchesToUpdate = append(branchesToUpdate, branchSettings)
			}
		}
	}

	for branchToDelete := range deleteBranchesMap {
		log.Printf("[INFO] Removing branch protection for %s\n", branchToDelete)

		_, err := client.github.Repositories.RemoveBranchProtection(context.Background(), owner, name, branchToDelete)

		if err != nil {
			return errors.Wrap(err, "Error removing branch protection\n")
		}
	}

	for _, branchSettings := range branchesToUpdate {
		log.Printf("[INFO] Updating branch protection for %s\n", branchSettings.Name)

		var requiredReviews *github.PullRequestReviewsEnforcementRequest

		if branchSettings.Protection.RequiredApprovingReviewCount.RequiredApprovingReviewCount == 0 {
			requiredReviews = nil
		} else {
			requiredReviews = &github.PullRequestReviewsEnforcementRequest{
				DismissStaleReviews:          branchSettings.Protection.RequiredApprovingReviewCount.DismissStaleReviews,
				RequireCodeOwnerReviews:      branchSettings.Protection.RequiredApprovingReviewCount.RequireCodeOwnerReviews,
				RequiredApprovingReviewCount: branchSettings.Protection.RequiredApprovingReviewCount.RequiredApprovingReviewCount,
			}
		}

		_, _, err := client.github.Repositories.UpdateBranchProtection(context.Background(), owner, name, branchSettings.Name, &github.ProtectionRequest{
			EnforceAdmins: branchSettings.Protection.EnforceAdmins,
			RequiredStatusChecks: &github.RequiredStatusChecks{
				Strict:   branchSettings.Protection.RequiredStatusChecks.Strict,
				Contexts: branchSettings.Protection.RequiredStatusChecks.Contexts,
			},
			RequiredPullRequestReviews: requiredReviews,
		})

		if err != nil {
			return errors.Wrap(err, "Error updating branch protection\n")
		}
	}

	return nil
}

func (client *Client) updateWebhooks(owner string, name string, githubWebhooks []webhook, webhooksSettings []webhook) error {
	webhooksToUpdate := []webhook{}
	deleteWebhooksMap := map[string]webhook{}

	for _, githubWebhook := range githubWebhooks {
		deleteWebhooksMap[githubWebhook.URL] = githubWebhook
	}

	for _, webhookSettings := range webhooksSettings {
		githubWebhook, ok := deleteWebhooksMap[webhookSettings.URL]

		if !ok {
			log.Printf("[INFO] Creating new webhook %s\n", webhookSettings.URL)

			_, _, err := client.github.Repositories.CreateHook(context.Background(), owner, name, &github.Hook{
				Events: webhookSettings.Events,
				Active: github.Bool(true),
				Config: map[string]interface{}{
					"content_type": webhookSettings.ContentType,
					"secret":       webhookSettings.Secret,
					"url":          webhookSettings.URL,
				},
			})

			if err != nil {
				return errors.Wrap(err, "Error creating webhook\n")
			}
		} else {
			delete(deleteWebhooksMap, webhookSettings.URL)

			webhookSettings.ID = githubWebhook.ID
			githubWebhook.Secret = webhookSettings.Secret

			if !reflect.DeepEqual(githubWebhook, webhookSettings) {
				webhooksToUpdate = append(webhooksToUpdate, webhookSettings)
			}
		}
	}

	for _, webhookToDelete := range deleteWebhooksMap {
		log.Printf("[INFO] Removing webhook %s\n", webhookToDelete.URL)

		_, err := client.github.Repositories.DeleteHook(context.Background(), owner, name, webhookToDelete.ID)

		if err != nil {
			return errors.Wrap(err, "Error removing webhook\n")
		}
	}

	for _, webhookToUpdate := range webhooksToUpdate {
		log.Printf("[INFO] Updating webhook %s\n", webhookToUpdate.URL)

		_, _, err := client.github.Repositories.EditHook(context.Background(), owner, name, webhookToUpdate.ID, &github.Hook{
			Events: webhookToUpdate.Events,
			Active: github.Bool(true),
			Config: map[string]interface{}{
				"content_type": webhookToUpdate.ContentType,
				"secret":       webhookToUpdate.Secret,
				"url":          webhookToUpdate.URL,
			},
		})

		if err != nil {
			return errors.Wrap(err, "Error updating webhook\n")
		}
	}

	return nil
}
