package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// InteractiveMode handles the interactive conversation session
type InteractiveMode struct {
	convService *ConversationService
	nlParser    *NLQueryParser
	bedrock     BedrockService
	owner       string
	repo        string
	releases    []Release
}

// NewInteractiveMode creates a new interactive session
func NewInteractiveMode(bedrock BedrockService, owner, repo string) *InteractiveMode {
	return &InteractiveMode{
		convService: NewConversationService(bedrock, owner, repo),
		nlParser:    NewNLQueryParser(bedrock),
		bedrock:     bedrock,
		owner:       owner,
		repo:        repo,
	}
}

// SetReleases caches the fetched releases for the session
func (im *InteractiveMode) SetReleases(releases []Release) {
	im.releases = releases
	im.convService.SetReleases(releases)
}

var interactiveSystemPrompt = `You are an AI assistant helping users explore GitHub release history for the {{.Repo}} repository owned by {{.Owner}}.

You have access to the following release data:
{{.ReleaseData}}

You can help users with:
1. Searching for specific changes or packages
2. Understanding what changed between versions
3. Finding breaking changes
4. Generating migration guides
5. Summarizing release trends

When analyzing releases:
- Be specific about version numbers and dates
- Quote relevant sections from release notes
- Highlight breaking changes with clear warnings
- Suggest migration steps when applicable

Current context:
- Repository: {{.Owner}}/{{.Repo}}
- Releases loaded: {{.ReleaseCount}}
- Date range: {{.DateRange}}

Respond conversationally but stay factual. If you don't have information about something, say so.`

// buildSystemPrompt creates the system prompt with release context
func (im *InteractiveMode) buildSystemPrompt() string {
	releaseData := Stringify(im.releases)

	// Get date range
	dateRange := "N/A"
	if len(im.releases) > 0 {
		oldest := im.releases[len(im.releases)-1].CreatedAt.Format("2006-01-02")
		newest := im.releases[0].CreatedAt.Format("2006-01-02")
		dateRange = fmt.Sprintf("%s to %s", oldest, newest)
	}

	return Prompt(interactiveSystemPrompt, struct {
		Owner        string
		Repo         string
		ReleaseData  string
		ReleaseCount int
		DateRange    string
	}{
		Owner:        im.owner,
		Repo:         im.repo,
		ReleaseData:  releaseData,
		ReleaseCount: len(im.releases),
		DateRange:    dateRange,
	})
}

// Run starts the interactive session
func (im *InteractiveMode) Run() error {
	// Set up the conversation with context
	im.convService.SetSystemPrompt(im.buildSystemPrompt())

	// Print welcome message
	im.printWelcome()

	// Main interaction loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle special commands
		if im.handleCommand(input) {
			continue
		}

		// Process the user's query
		if err := im.processQuery(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// handleCommand processes special commands, returns true if it was a command
func (im *InteractiveMode) handleCommand(input string) bool {
	cmd := strings.ToLower(strings.TrimSpace(input))

	switch {
	case cmd == "/help" || cmd == "help":
		im.printHelp()
		return true

	case cmd == "/quit" || cmd == "/exit" || cmd == "quit" || cmd == "exit":
		fmt.Println("\nGoodbye!")
		os.Exit(0)
		return true

	case cmd == "/clear":
		im.convService.ClearHistory()
		fmt.Println("Conversation history cleared.")
		return true

	case cmd == "/releases":
		im.printReleaseList()
		return true

	case cmd == "/status":
		im.printStatus()
		return true

	case strings.HasPrefix(cmd, "/breaking"):
		return im.handleBreakingCommand(input)

	case strings.HasPrefix(cmd, "/search"):
		keyword := strings.TrimPrefix(input, "/search ")
		keyword = strings.TrimSpace(keyword)
		if keyword != "" && keyword != "/search" {
			im.handleSearchCommand(keyword)
		} else {
			fmt.Println("Usage: /search <keyword>")
		}
		return true
	}

	return false
}

// handleBreakingCommand handles the /breaking command
func (im *InteractiveMode) handleBreakingCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) < 3 {
		fmt.Println("Usage: /breaking <from_version> <to_version>")
		fmt.Println("Example: /breaking v1.0.0 v2.0.0")
		return true
	}

	fromVersion := parts[1]
	toVersion := parts[2]

	fmt.Printf("\nAnalyzing breaking changes from %s to %s...\n", fromVersion, toVersion)

	agent := NewBreakingChangeAgent(im.bedrock, im.owner, im.repo)
	agent.SetVersionRange(fromVersion, toVersion)

	report, err := agent.Run(im.releases)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return true
	}

	fmt.Print(FormatReport(report))
	return true
}

// handleSearchCommand handles the /search command
func (im *InteractiveMode) handleSearchCommand(keyword string) {
	fmt.Printf("\nSearching for '%s' in releases...\n\n", keyword)

	filter := NewSemanticFilter(SimpleFilter{})
	matches := filter.Filter(im.releases, keyword)

	if len(matches) == 0 {
		fmt.Printf("No releases found containing '%s'\n", keyword)
		return
	}

	fmt.Printf("Found %d releases containing '%s':\n\n", len(matches), keyword)
	for _, r := range matches {
		fmt.Printf("  • %s (%s)\n", r.TagName, r.CreatedAt.Format("2006-01-02"))
		fmt.Printf("    %s\n\n", r.HTMLURL)
	}
}

// processQuery sends a query to the conversation service
func (im *InteractiveMode) processQuery(query string) error {
	// Check for special query patterns
	intent, _ := im.nlParser.Parse(query)
	if intent != nil {
		switch intent.QueryType {
		case "breaking_changes":
			if intent.FromVersion != "" && intent.ToVersion != "" {
				fmt.Printf("\nDetected breaking change query. Analyzing %s to %s...\n",
					intent.FromVersion, intent.ToVersion)

				agent := NewBreakingChangeAgent(im.bedrock, im.owner, im.repo)
				agent.SetVersionRange(intent.FromVersion, intent.ToVersion)

				report, err := agent.Run(im.releases)
				if err != nil {
					return err
				}

				fmt.Print(FormatReport(report))
				return nil
			}

		case "migration":
			if intent.FromVersion != "" && intent.ToVersion != "" {
				fmt.Printf("\nGenerating migration guide from %s to %s...\n",
					intent.FromVersion, intent.ToVersion)

				agent := NewBreakingChangeAgent(im.bedrock, im.owner, im.repo)
				agent.SetVersionRange(intent.FromVersion, intent.ToVersion)

				report, err := agent.Run(im.releases)
				if err != nil {
					return err
				}

				if report.MigrationGuide != "" {
					fmt.Println("\n" + report.MigrationGuide)
				} else {
					fmt.Println("\nNo migration steps needed - no breaking changes detected.")
				}
				return nil
			}
		}
	}

	// Regular conversation
	fmt.Println("\nThinking...")

	response, err := im.convService.Chat(query)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n", response)
	return nil
}

// printWelcome prints the welcome message
func (im *InteractiveMode) printWelcome() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           GitHub Release Finder - Interactive Mode            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Repository: %s/%s\n", im.owner, im.repo)
	fmt.Printf("  Releases:   %d loaded\n", len(im.releases))
	fmt.Println()
	fmt.Println("  Type your questions about releases in natural language.")
	fmt.Println("  Type /help for available commands.")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    • What changed in the last 5 releases?")
	fmt.Println("    • Are there any breaking changes between v1.0 and v2.0?")
	fmt.Println("    • Show me all S3-related changes")
	fmt.Println("    • Generate a migration guide from v1.5 to v2.0")
	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
}

// printHelp prints available commands
func (im *InteractiveMode) printHelp() {
	fmt.Println()
	fmt.Println("AVAILABLE COMMANDS:")
	fmt.Println()
	fmt.Println("  /help              - Show this help message")
	fmt.Println("  /quit, /exit       - Exit interactive mode")
	fmt.Println("  /clear             - Clear conversation history")
	fmt.Println("  /releases          - List all loaded releases")
	fmt.Println("  /status            - Show current session status")
	fmt.Println("  /search <keyword>  - Quick search for keyword in releases")
	fmt.Println("  /breaking <from> <to> - Analyze breaking changes between versions")
	fmt.Println()
	fmt.Println("NATURAL LANGUAGE QUERIES:")
	fmt.Println()
	fmt.Println("  You can also ask questions naturally:")
	fmt.Println("    • What security fixes were in the recent releases?")
	fmt.Println("    • Summarize the changes in v2.0.0")
	fmt.Println("    • What packages were updated?")
	fmt.Println("    • Are there any deprecations I should know about?")
	fmt.Println()
}

// printReleaseList prints a list of loaded releases
func (im *InteractiveMode) printReleaseList() {
	fmt.Println()
	fmt.Println("LOADED RELEASES:")
	fmt.Println()
	for i, r := range im.releases {
		status := ""
		if r.Prerelease {
			status = " [pre-release]"
		}
		fmt.Printf("  %2d. %s (%s)%s\n", i+1, r.TagName, r.CreatedAt.Format("2006-01-02"), status)
	}
	fmt.Printf("\nTotal: %d releases\n", len(im.releases))
}

// printStatus prints the current session status
func (im *InteractiveMode) printStatus() {
	fmt.Println()
	fmt.Println("SESSION STATUS:")
	fmt.Println()
	fmt.Printf("  Repository:    %s/%s\n", im.owner, im.repo)
	fmt.Printf("  Releases:      %d loaded\n", len(im.releases))
	fmt.Printf("  Messages:      %d in conversation\n", len(im.convService.GetConversation().GetMessages()))

	if len(im.releases) > 0 {
		fmt.Printf("  Latest:        %s (%s)\n",
			im.releases[0].TagName,
			im.releases[0].CreatedAt.Format("2006-01-02"))
		fmt.Printf("  Oldest:        %s (%s)\n",
			im.releases[len(im.releases)-1].TagName,
			im.releases[len(im.releases)-1].CreatedAt.Format("2006-01-02"))
	}
	fmt.Println()
}

// RunNonInteractive processes a single natural language query
func RunNonInteractive(bedrock BedrockService, query string, releases []Release, owner, repo string) error {
	slog.Info("Processing natural language query", "query", query)

	parser := NewNLQueryParser(bedrock)
	intent, err := parser.Parse(query)
	if err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	// Enrich intent with provided values
	parser.EnrichIntent(intent, owner, repo, len(releases))

	slog.Info("Query intent parsed",
		"type", intent.QueryType,
		"owner", intent.Owner,
		"repo", intent.Repository,
		"keywords", intent.Keywords)

	switch intent.QueryType {
	case "breaking_changes", "migration":
		agent := NewBreakingChangeAgent(bedrock, owner, repo)
		agent.SetVersionRange(intent.FromVersion, intent.ToVersion)

		report, err := agent.Run(releases)
		if err != nil {
			return err
		}

		fmt.Print(FormatReport(report))

		if intent.QueryType == "migration" && report.MigrationGuide != "" {
			fmt.Println("\n" + report.MigrationGuide)
		}

	case "compare":
		// Handle version comparison
		agent := NewBreakingChangeAgent(bedrock, owner, repo)
		agent.SetVersionRange(intent.FromVersion, intent.ToVersion)

		report, err := agent.Run(releases)
		if err != nil {
			return err
		}

		fmt.Print(FormatReport(report))

	default:
		// Regular search - use conversation service for single query
		conv := NewConversationService(bedrock, owner, repo)
		conv.SetReleases(releases)

		systemPrompt := Prompt(interactiveSystemPrompt, struct {
			Owner        string
			Repo         string
			ReleaseData  string
			ReleaseCount int
			DateRange    string
		}{
			Owner:        owner,
			Repo:         repo,
			ReleaseData:  Stringify(releases),
			ReleaseCount: len(releases),
			DateRange:    getDateRange(releases),
		})
		conv.SetSystemPrompt(systemPrompt)

		response, err := conv.Chat(query)
		if err != nil {
			return err
		}

		fmt.Println(response)
	}

	return nil
}

func getDateRange(releases []Release) string {
	if len(releases) == 0 {
		return "N/A"
	}
	oldest := releases[len(releases)-1].CreatedAt.Format("2006-01-02")
	newest := releases[0].CreatedAt.Format("2006-01-02")
	return fmt.Sprintf("%s to %s", oldest, newest)
}
