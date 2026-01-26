package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

var (
	baseURL        = "https://api.github.com"
	awsRegion      = "us-east-1"
	runtimeVersion = "bedrock-2023-05-31"
	maxTokens      = 4096
	claudeModel    = "anthropic.claude-opus-4-20250514-v1:0"
)

func main() {
	options := cliHandler()

	// Initialize the Bedrock service early for NL parsing if needed
	llmService, err := NewBedRockService(awsRegion, Claude{
		model:   claudeModel,
		version: runtimeVersion,
	})
	if err != nil {
		slog.Error("Failed to initialize AWS Bedrock service", "error", err)
		fmt.Println("Error: Could not connect to AWS Bedrock. Please check your AWS credentials.")
		os.Exit(1)
	}

	// Handle natural language queries that might contain owner/repo
	if options.Query != "" && (options.RepoOwner == "" || options.Repo == "") {
		parser := NewNLQueryParser(llmService)
		intent, err := parser.Parse(options.Query)
		if err != nil {
			slog.Error("Failed to parse natural language query", "error", err)
			fmt.Println("Error: Could not understand the query. Please specify -o (owner) and -r (repo) flags.")
			os.Exit(1)
		}

		// Fill in missing values from parsed intent
		if options.RepoOwner == "" && intent.Owner != "" {
			options.RepoOwner = intent.Owner
		}
		if options.Repo == "" && intent.Repository != "" {
			options.Repo = intent.Repository
		}
		if intent.NumReleases > 0 && intent.NumReleases != 20 {
			options.NumRecords = intent.NumReleases
		}

		// Update query intent for later use
		if intent.QueryType == "breaking_changes" || intent.QueryType == "migration" {
			if options.FromVersion == "" {
				options.FromVersion = intent.FromVersion
			}
			if options.ToVersion == "" {
				options.ToVersion = intent.ToVersion
			}
		}
	}

	// Validate required fields
	if err := options.Validate(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nUse -h or --help for usage information.")
		os.Exit(1)
	}

	// Build the GitHub API URL and fetch releases
	url := listReleasesURL(options.RepoOwner, options.Repo, options.NumRecords)
	slog.Info("Fetching releases", "url", url)

	releaseHistory, err := Get(url, ListReleasesParser)
	if err != nil {
		slog.Error("GitHub API call failed", "error", err)
		fmt.Printf("Error: Could not fetch releases from GitHub: %v\n", err)
		os.Exit(1)
	}

	if len(releaseHistory.Releases) == 0 {
		fmt.Printf("No releases found for %s/%s. The repository may not publish releases.\n",
			options.RepoOwner, options.Repo)
		os.Exit(0)
	}

	slog.Info("Releases fetched", "count", len(releaseHistory.Releases))

	// Route to appropriate mode
	mode := options.Mode()
	slog.Info("Operating mode", "mode", mode)

	switch mode {
	case "interactive":
		runInteractiveMode(llmService, options, releaseHistory.Releases)

	case "breaking_changes":
		runBreakingChangeMode(llmService, options, releaseHistory.Releases)

	case "natural_language":
		runNaturalLanguageMode(llmService, options, releaseHistory.Releases)

	case "keyword_search":
		runKeywordSearchMode(llmService, options, releaseHistory.Releases)

	default:
		runInteractiveMode(llmService, options, releaseHistory.Releases)
	}
}

// runInteractiveMode starts the interactive conversation session
func runInteractiveMode(llmService BedrockService, options CommandLine, releases []Release) {
	interactive := NewInteractiveMode(llmService, options.RepoOwner, options.Repo)
	interactive.SetReleases(releases)

	if err := interactive.Run(); err != nil {
		slog.Error("Interactive mode error", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// runBreakingChangeMode runs the breaking change detection agent
func runBreakingChangeMode(llmService BedrockService, options CommandLine, releases []Release) {
	fmt.Printf("Analyzing breaking changes from %s to %s in %s/%s...\n\n",
		options.FromVersion, options.ToVersion, options.RepoOwner, options.Repo)

	agent := NewBreakingChangeAgent(llmService, options.RepoOwner, options.Repo)
	agent.SetVersionRange(options.FromVersion, options.ToVersion)

	report, err := agent.Run(releases)
	if err != nil {
		slog.Error("Breaking change analysis failed", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if options.JSONOutput {
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			slog.Error("Failed to marshal report to JSON", "error", err)
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Print(FormatReport(report))
	}
}

// runNaturalLanguageMode processes a natural language query
func runNaturalLanguageMode(llmService BedrockService, options CommandLine, releases []Release) {
	// Check if this looks like a breaking change query
	if options.FromVersion != "" && options.ToVersion != "" {
		runBreakingChangeMode(llmService, options, releases)
		return
	}

	if err := RunNonInteractive(llmService, options.Query, releases, options.RepoOwner, options.Repo); err != nil {
		slog.Error("Query processing failed", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// runKeywordSearchMode runs the legacy keyword search
func runKeywordSearchMode(llmService BedrockService, options CommandLine, releases []Release) {
	// Filter releases by keyword
	filter := NewSemanticFilter(SimpleFilter{})
	filteredReleases := filter.Filter(releases, options.SearchTerm)

	if len(filteredReleases) == 0 {
		fmt.Printf("No releases found containing '%s'.\n", options.SearchTerm)
		fmt.Println("If this was unexpected, verify that the repository publishes detailed release notes.")
		os.Exit(0)
	}

	slog.Info("Filtered releases", "count", len(filteredReleases), "keyword", options.SearchTerm)

	// Use Claude to analyze the filtered releases
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
		slog.Error("Failed to analyze releases", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if options.JSONOutput {
		// Try to parse as JSON and pretty print
		for _, content := range resp {
			var jsonData interface{}
			if err := json.Unmarshal([]byte(content.Text), &jsonData); err == nil {
				prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
				fmt.Println(string(prettyJSON))
			} else {
				fmt.Println(content.Text)
			}
		}
	} else {
		for _, content := range resp {
			fmt.Println(content.Text)
		}
	}
}
