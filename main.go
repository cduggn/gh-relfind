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
	claudeModel    = "anthropic.claude-3-sonnet-20240229-v1"
)

func main() {
	options := cliHandler()

	url := createReleaseHistoryURL(options.RepoOwner, options.Repo, options.NumRecords)
	slog.Info(fmt.Sprintf("URL: %v", url))

	releaseHistory, err := Get(url, parseListReleases)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get release history: %v ", err))
		os.Exit(1)
	}

	llmService, err := NewBedRockService(awsRegion)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create client: %v ", err))
		os.Exit(1)
	}

	llmService.SetChatModel(Claude{
		model:   claudeModel,
		version: runtimeVersion,
	})

	promptData := Stringify(*releaseHistory)

	resp, err := llmService.Chat(
		Prompt(claudeUserPrompt,
			struct{ Keyword string }{Keyword: "cost explorer"}),
		Prompt(claudeSystemPrompt, struct {
			Repo    string
			History string
		}{
			Repo:    options.Repo,
			History: promptData,
		}))

	if err != nil {
		slog.Error(fmt.Sprintf("Failed to get response: %v ", err))
		os.Exit(1)
	}

	fmt.Println(resp)

	//llmService.GenerateEmbeddings(Bedrock{})
}
