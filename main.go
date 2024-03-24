package main

import (
	"fmt"
	"log/slog"
	"os"
)

var (
	baseURL        = "https://api.github.com"
	awsRegion      = "us-east-1"
	modelId        = "anthropic.claude-3-sonnet-20240229-v1:0"
	runtimeVersion = "bedrock-2023-05-31"
	maxTokens      = 500
	claudeModel    = "anthropic.claude-3-sonnet-20240229-v1:0"
)

func main() {
	options := cliHandler()

	// dynamically create the list releases URL
	url := listReleasesURL(options.RepoOwner, options.Repo, options.NumRecords)
	slog.Info(fmt.Sprintf("List Releases URL: %v", url))

	// get the list of releases from the GitHub API
	releaseHistory, err := Get(url, ListReleasesParser)
	if err != nil {
		slog.Error(fmt.Sprintf("github list releases API call failed: %v ", err))
		os.Exit(1)
	}

	// filter the releases based on the search term
	s := NewSemanticFilter(SimpleFilter{})
	filteredReleases := s.Filter(releaseHistory.Releases, options.SearchTerm)

	if len(filteredReleases) == 0 {
		slog.Info(fmt.Sprintf("No releases found for search term: %v. If this was unexpected verify that the repository publishes release information.", options.SearchTerm))
		os.Exit(0)
	}

	// instantiate the bedrock service
	llmService, err := NewBedRockService(awsRegion, Claude{
		model:   claudeModel,
		version: runtimeVersion,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("aws bedrock service instantiation failed: %v ", err))
		os.Exit(1)
	}

	// chat with the model
	resp, err := llmService.Chat(
		Prompt(claudeUserPrompt,
			struct{ Keyword string }{Keyword: options.SearchTerm}),
		Prompt(claudeSystemPrompt, struct {
			Repo    string
			History string
		}{
			Repo:    options.Repo,
			History: Stringify(filteredReleases),
		}))

	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get response: %v ", err))
		os.Exit(1)
	}

	// print the response
	for _, content := range resp {
		fmt.Println(content.Text)
	}
}
