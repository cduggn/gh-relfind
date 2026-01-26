package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// CommandLine holds all CLI configuration
type CommandLine struct {
	// Repository settings
	Repo       string
	RepoOwner  string
	NumRecords int

	// Mode flags
	Interactive bool   // -i: Interactive conversation mode
	Query       string // Natural language query (positional or -q)

	// Legacy flags (still supported)
	SearchTerm string // -k: Keyword search (legacy mode)

	// Breaking change detection
	FromVersion string // --from: Starting version for breaking change detection
	ToVersion   string // --to: Ending version for breaking change detection

	// Output options
	JSONOutput bool // --json: Output in JSON format
}

func cliHandler() CommandLine {
	// Repository settings
	repoPtr := flag.String("r", "", "Repository name (e.g., aws-sdk-go-v2)")
	repoOwnerPtr := flag.String("o", "", "Repository owner (e.g., aws)")
	numRecordsPtr := flag.Int("n", 20, "Number of releases to fetch (default: 20, max: 100)")

	// Mode flags
	interactivePtr := flag.Bool("i", false, "Start interactive conversation mode")
	queryPtr := flag.String("q", "", "Natural language query")

	// Legacy support
	searchTermPtr := flag.String("k", "", "Keyword search (legacy mode)")

	// Breaking change detection
	fromVersionPtr := flag.String("from", "", "Starting version for breaking change analysis")
	toVersionPtr := flag.String("to", "", "Ending version for breaking change analysis")

	// Output options
	jsonOutputPtr := flag.Bool("json", false, "Output results in JSON format")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
gh-relfind - GitHub Release Finder with AI-powered analysis

USAGE:
  gh-relfind [options] [natural language query]

MODES:
  Interactive Mode (-i):
    Start a conversation session to explore releases interactively.

    Example:
      gh-relfind -o aws -r aws-sdk-go-v2 -i

  Natural Language Query:
    Ask questions about releases in plain English.

    Examples:
      gh-relfind -o aws -r aws-sdk-go-v2 "What S3 changes happened recently?"
      gh-relfind -o aws -r aws-sdk-go-v2 -q "Show me breaking changes between v1.20 and v1.30"
      gh-relfind "Show me security fixes in kubernetes/kubernetes from the last 50 releases"

  Breaking Change Detection (--from, --to):
    Analyze breaking changes between two versions.

    Example:
      gh-relfind -o aws -r aws-sdk-go-v2 --from v1.20.0 --to v1.30.0

  Keyword Search (-k, legacy):
    Simple keyword-based filtering (original behavior).

    Example:
      gh-relfind -o aws -r aws-sdk-go-v2 -k costexplorer -n 30

OPTIONS:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
EXAMPLES:
  # Interactive mode
  gh-relfind -o kubernetes -r kubernetes -n 50 -i

  # Natural language query
  gh-relfind -o aws -r aws-sdk-go-v2 "What changed in DynamoDB?"

  # Shorthand with owner/repo in query
  gh-relfind "Show me S3 changes in aws/aws-sdk-go-v2"

  # Breaking change detection
  gh-relfind -o hashicorp -r terraform --from v1.5.0 --to v1.6.0

  # Legacy keyword search
  gh-relfind -o aws -r aws-sdk-go-v2 -k s3 -n 20

  # JSON output
  gh-relfind -o aws -r aws-sdk-go-v2 --from v1.0.0 --to v2.0.0 --json

`)
	}

	flag.Parse()

	// Get positional arguments (natural language query)
	args := flag.Args()
	query := *queryPtr
	if query == "" && len(args) > 0 {
		query = strings.Join(args, " ")
	}

	// Validate num records
	numRecords := *numRecordsPtr
	if numRecords > 100 {
		numRecords = 100
	}
	if numRecords < 1 {
		numRecords = 20
	}

	return CommandLine{
		Repo:        *repoPtr,
		RepoOwner:   *repoOwnerPtr,
		NumRecords:  numRecords,
		Interactive: *interactivePtr,
		Query:       query,
		SearchTerm:  *searchTermPtr,
		FromVersion: *fromVersionPtr,
		ToVersion:   *toVersionPtr,
		JSONOutput:  *jsonOutputPtr,
	}
}

// Validate checks if the command line options are valid
func (c *CommandLine) Validate() error {
	// If we have a query, we might extract owner/repo from it
	if c.Query != "" {
		return nil // Will be validated after NL parsing
	}

	// For non-query modes, we need owner and repo
	if c.RepoOwner == "" {
		return fmt.Errorf("repository owner is required (-o flag)")
	}
	if c.Repo == "" {
		return fmt.Errorf("repository name is required (-r flag)")
	}

	return nil
}

// Mode returns the detected operation mode
func (c *CommandLine) Mode() string {
	if c.Interactive {
		return "interactive"
	}
	if c.FromVersion != "" && c.ToVersion != "" {
		return "breaking_changes"
	}
	if c.Query != "" {
		return "natural_language"
	}
	if c.SearchTerm != "" {
		return "keyword_search"
	}
	return "interactive" // Default to interactive if no specific mode
}
