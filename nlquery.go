package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// QueryIntent represents the structured intent extracted from natural language
type QueryIntent struct {
	QueryType    string   `json:"query_type"`    // "search", "compare", "breaking_changes", "migration"
	Owner        string   `json:"owner"`         // Repository owner
	Repository   string   `json:"repository"`    // Repository name
	Keywords     []string `json:"keywords"`      // Search keywords
	FromVersion  string   `json:"from_version"`  // Starting version for comparisons
	ToVersion    string   `json:"to_version"`    // Ending version for comparisons
	NumReleases  int      `json:"num_releases"`  // Number of releases to fetch
	TimeRange    string   `json:"time_range"`    // e.g., "last 3 months", "since 2024"
	RawQuery     string   `json:"raw_query"`     // Original query text
}

// NLQueryParser parses natural language queries into structured intents
type NLQueryParser struct {
	bedrock BedrockService
}

// NewNLQueryParser creates a new natural language query parser
func NewNLQueryParser(bedrock BedrockService) *NLQueryParser {
	return &NLQueryParser{bedrock: bedrock}
}

var nlParserSystemPrompt = `You are a query parser for a GitHub release search tool. Your job is to extract structured information from natural language queries about GitHub releases.

Parse the user's query and return a JSON object with these fields:
- query_type: One of "search" (general search), "compare" (version comparison), "breaking_changes" (find breaking changes), "migration" (upgrade guide)
- owner: GitHub repository owner (if mentioned)
- repository: GitHub repository name (if mentioned)
- keywords: Array of search keywords/terms mentioned
- from_version: Starting version for comparisons (if mentioned, e.g., "v1.2.0")
- to_version: Ending version for comparisons (if mentioned, e.g., "v2.0.0")
- num_releases: Number of releases to search (default to 20 if not specified, max 100)
- time_range: Time constraint if mentioned (e.g., "last 3 months", "since January 2024")

Rules:
1. Extract repository info from patterns like "owner/repo", "in repo X", "for X repository"
2. Look for version patterns like "v1.2.3", "1.2.3", "version 1.2"
3. Identify breaking change queries from words like "breaking", "migration", "upgrade", "incompatible"
4. Default num_releases to 20 if not specified
5. Return ONLY valid JSON, no explanations

Examples:
Query: "Show me S3 changes in aws/aws-sdk-go-v2 from the last 30 releases"
{"query_type":"search","owner":"aws","repository":"aws-sdk-go-v2","keywords":["S3"],"from_version":"","to_version":"","num_releases":30,"time_range":""}

Query: "What breaking changes between v1.20 and v1.30 in kubernetes/kubernetes"
{"query_type":"breaking_changes","owner":"kubernetes","repository":"kubernetes","keywords":[],"from_version":"v1.20","to_version":"v1.30","num_releases":50,"time_range":""}

Query: "Generate migration guide for upgrading from 2.0 to 3.0"
{"query_type":"migration","owner":"","repository":"","keywords":[],"from_version":"2.0","to_version":"3.0","num_releases":50,"time_range":""}`

// Parse takes a natural language query and returns a structured QueryIntent
func (p *NLQueryParser) Parse(query string) (*QueryIntent, error) {
	// First try simple regex parsing for common patterns
	if intent := p.quickParse(query); intent != nil {
		return intent, nil
	}

	// Fall back to LLM parsing for complex queries
	return p.llmParse(query)
}

// quickParse attempts to parse simple, well-structured queries without LLM
func (p *NLQueryParser) quickParse(query string) *QueryIntent {
	intent := &QueryIntent{
		QueryType:   "search",
		NumReleases: 20,
		RawQuery:    query,
	}

	// Extract owner/repo pattern
	repoPattern := regexp.MustCompile(`([a-zA-Z0-9_-]+)/([a-zA-Z0-9_.-]+)`)
	if matches := repoPattern.FindStringSubmatch(query); len(matches) == 3 {
		intent.Owner = matches[1]
		intent.Repository = matches[2]
	}

	// Extract number of releases
	numPattern := regexp.MustCompile(`(?:last\s+)?(\d+)\s+releases?`)
	if matches := numPattern.FindStringSubmatch(query); len(matches) == 2 {
		fmt.Sscanf(matches[1], "%d", &intent.NumReleases)
	}

	// Extract version patterns
	versionPattern := regexp.MustCompile(`v?(\d+\.\d+(?:\.\d+)?)`)
	versions := versionPattern.FindAllString(query, -1)
	if len(versions) >= 2 {
		intent.FromVersion = versions[0]
		intent.ToVersion = versions[1]
	}

	// Detect query type from keywords
	lowerQuery := strings.ToLower(query)
	if strings.Contains(lowerQuery, "breaking") || strings.Contains(lowerQuery, "incompatible") {
		intent.QueryType = "breaking_changes"
	} else if strings.Contains(lowerQuery, "migration") || strings.Contains(lowerQuery, "upgrade guide") {
		intent.QueryType = "migration"
	} else if strings.Contains(lowerQuery, "compare") || strings.Contains(lowerQuery, "between") {
		intent.QueryType = "compare"
	}

	// Only return quick parse result if we have repo info
	if intent.Owner != "" && intent.Repository != "" {
		return intent
	}

	return nil
}

// llmParse uses Claude to parse complex natural language queries
func (p *NLQueryParser) llmParse(query string) (*QueryIntent, error) {
	resp, err := p.bedrock.Chat(query, nlParserSystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query with LLM: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract JSON from response (handle potential markdown code blocks)
	jsonStr := resp[0].Text
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	var intent QueryIntent
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w, response: %s", err, jsonStr)
	}

	intent.RawQuery = query

	// Apply defaults
	if intent.NumReleases == 0 {
		intent.NumReleases = 20
	}
	if intent.NumReleases > 100 {
		intent.NumReleases = 100
	}

	return &intent, nil
}

// ValidateIntent checks if the intent has minimum required fields
func (p *NLQueryParser) ValidateIntent(intent *QueryIntent) error {
	if intent.Owner == "" {
		return fmt.Errorf("repository owner not specified. Use format: owner/repo or specify with -o flag")
	}
	if intent.Repository == "" {
		return fmt.Errorf("repository name not specified. Use format: owner/repo or specify with -r flag")
	}
	return nil
}

// EnrichIntent fills in missing fields from CLI flags
func (p *NLQueryParser) EnrichIntent(intent *QueryIntent, owner, repo string, numReleases int) {
	if intent.Owner == "" && owner != "" {
		intent.Owner = owner
	}
	if intent.Repository == "" && repo != "" {
		intent.Repository = repo
	}
	if intent.NumReleases == 0 || (numReleases != 10 && numReleases > 0) {
		intent.NumReleases = numReleases
	}
}
