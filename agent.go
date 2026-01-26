package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

// AgentState represents the current state of the agent
type AgentState string

const (
	StateInitial           AgentState = "initial"
	StateFetchingReleases  AgentState = "fetching_releases"
	StateAnalyzing         AgentState = "analyzing"
	StateGeneratingReport  AgentState = "generating_report"
	StateComplete          AgentState = "complete"
	StateError             AgentState = "error"
)

// BreakingChange represents a detected breaking change
type BreakingChange struct {
	Version     string `json:"version"`
	Package     string `json:"package"`
	Type        string `json:"type"` // "api", "behavior", "deprecation", "removal", "dependency"
	Description string `json:"description"`
	Severity    string `json:"severity"` // "high", "medium", "low"
	Migration   string `json:"migration"`
	URL         string `json:"url"`
}

// BreakingChangeReport is the full analysis report
type BreakingChangeReport struct {
	FromVersion    string           `json:"from_version"`
	ToVersion      string           `json:"to_version"`
	Repository     string           `json:"repository"`
	TotalReleases  int              `json:"total_releases"`
	BreakingCount  int              `json:"breaking_count"`
	HighSeverity   int              `json:"high_severity"`
	MediumSeverity int              `json:"medium_severity"`
	LowSeverity    int              `json:"low_severity"`
	Changes        []BreakingChange `json:"changes"`
	Summary        string           `json:"summary"`
	MigrationGuide string           `json:"migration_guide,omitempty"`
}

// BreakingChangeAgent is an agent that detects breaking changes between versions
type BreakingChangeAgent struct {
	bedrock     BedrockService
	state       AgentState
	releases    []Release
	report      *BreakingChangeReport
	owner       string
	repo        string
	fromVersion string
	toVersion   string
}

// NewBreakingChangeAgent creates a new breaking change detection agent
func NewBreakingChangeAgent(bedrock BedrockService, owner, repo string) *BreakingChangeAgent {
	return &BreakingChangeAgent{
		bedrock: bedrock,
		state:   StateInitial,
		owner:   owner,
		repo:    repo,
	}
}

// SetVersionRange sets the version range to analyze
func (a *BreakingChangeAgent) SetVersionRange(from, to string) {
	a.fromVersion = normalizeVersion(from)
	a.toVersion = normalizeVersion(to)
}

// normalizeVersion ensures versions have consistent format
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// Run executes the agent workflow
func (a *BreakingChangeAgent) Run(releases []Release) (*BreakingChangeReport, error) {
	slog.Info("Breaking Change Agent started", "from", a.fromVersion, "to", a.toVersion)

	a.state = StateFetchingReleases

	// Filter releases to the version range
	filteredReleases := a.filterReleasesByVersion(releases)
	if len(filteredReleases) == 0 {
		return nil, fmt.Errorf("no releases found between %s and %s", a.fromVersion, a.toVersion)
	}

	a.releases = filteredReleases
	slog.Info("Filtered releases", "count", len(filteredReleases))

	a.state = StateAnalyzing

	// Analyze releases for breaking changes
	changes, err := a.analyzeReleases()
	if err != nil {
		a.state = StateError
		return nil, fmt.Errorf("failed to analyze releases: %w", err)
	}

	a.state = StateGeneratingReport

	// Generate the report
	report := a.generateReport(changes)
	a.report = report

	// Generate migration guide if there are breaking changes
	if len(changes) > 0 {
		guide, err := a.generateMigrationGuide(changes)
		if err != nil {
			slog.Warn("Failed to generate migration guide", "error", err)
		} else {
			report.MigrationGuide = guide
		}
	}

	a.state = StateComplete
	slog.Info("Breaking Change Agent completed", "breaking_changes", len(changes))

	return report, nil
}

// filterReleasesByVersion filters releases to those between fromVersion and toVersion
func (a *BreakingChangeAgent) filterReleasesByVersion(releases []Release) []Release {
	var filtered []Release

	// Sort releases by date (newest first)
	sorted := make([]Release, len(releases))
	copy(sorted, releases)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})

	inRange := false
	for _, r := range sorted {
		tag := r.TagName

		// Check if we've reached the end version (newer)
		if a.toVersion != "" && (tag == a.toVersion || strings.HasPrefix(tag, a.toVersion)) {
			inRange = true
		}

		if inRange {
			filtered = append(filtered, r)
		}

		// Check if we've passed the start version (older)
		if a.fromVersion != "" && (tag == a.fromVersion || strings.HasPrefix(tag, a.fromVersion)) {
			break
		}
	}

	// If no version range specified, return all
	if a.fromVersion == "" && a.toVersion == "" {
		return releases
	}

	// If we couldn't find version boundaries, return all releases
	// (better to analyze more than miss something)
	if len(filtered) == 0 {
		slog.Warn("Could not determine version boundaries, analyzing all releases")
		return releases
	}

	return filtered
}

var breakingChangeAnalysisPrompt = `You are an expert at analyzing software release notes for breaking changes.

Analyze the following GitHub release notes and identify ALL breaking changes. For each breaking change, provide:
1. The package/component affected
2. The type of change: "api" (signature changes), "behavior" (different behavior), "deprecation" (deprecated feature), "removal" (removed feature), "dependency" (dependency changes)
3. A clear description of what changed
4. The severity: "high" (requires immediate code changes), "medium" (may require changes), "low" (minimal impact)
5. Migration steps to address the change

Return your analysis as a JSON array of breaking changes. If no breaking changes are found, return an empty array [].

Format:
[
  {
    "package": "package name",
    "type": "api|behavior|deprecation|removal|dependency",
    "description": "what changed",
    "severity": "high|medium|low",
    "migration": "how to migrate"
  }
]

Be thorough. Look for:
- API signature changes
- Removed or renamed functions/methods
- Changed default behaviors
- Deprecated features
- Required parameter changes
- Return type changes
- Error handling changes
- Configuration changes
- Dependency version requirements

Release notes to analyze:
`

// analyzeReleases analyzes each release for breaking changes
func (a *BreakingChangeAgent) analyzeReleases() ([]BreakingChange, error) {
	var allChanges []BreakingChange

	for _, release := range a.releases {
		if release.Body == "" {
			continue
		}

		changes, err := a.analyzeRelease(release)
		if err != nil {
			slog.Warn("Failed to analyze release", "version", release.TagName, "error", err)
			continue
		}

		allChanges = append(allChanges, changes...)
	}

	return allChanges, nil
}

// analyzeRelease analyzes a single release for breaking changes
func (a *BreakingChangeAgent) analyzeRelease(release Release) ([]BreakingChange, error) {
	prompt := breakingChangeAnalysisPrompt + "\n\nVersion: " + release.TagName + "\n" + release.Body

	resp, err := a.bedrock.Chat(prompt, "You are a software breaking change detection assistant. Return only valid JSON arrays.")
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, nil
	}

	// Parse the response
	jsonStr := resp[0].Text
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	// Handle "no breaking changes" responses
	if jsonStr == "[]" || strings.Contains(strings.ToLower(jsonStr), "no breaking changes") {
		return nil, nil
	}

	var changes []BreakingChange
	if err := json.Unmarshal([]byte(jsonStr), &changes); err != nil {
		// Try to extract JSON from the response
		start := strings.Index(jsonStr, "[")
		end := strings.LastIndex(jsonStr, "]")
		if start >= 0 && end > start {
			jsonStr = jsonStr[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &changes); err != nil {
				slog.Warn("Failed to parse breaking changes JSON", "version", release.TagName)
				return nil, nil
			}
		} else {
			return nil, nil
		}
	}

	// Add version and URL to each change
	for i := range changes {
		changes[i].Version = release.TagName
		changes[i].URL = release.HTMLURL
	}

	return changes, nil
}

// generateReport creates a comprehensive report
func (a *BreakingChangeAgent) generateReport(changes []BreakingChange) *BreakingChangeReport {
	report := &BreakingChangeReport{
		FromVersion:   a.fromVersion,
		ToVersion:     a.toVersion,
		Repository:    a.owner + "/" + a.repo,
		TotalReleases: len(a.releases),
		BreakingCount: len(changes),
		Changes:       changes,
	}

	// Count by severity
	for _, c := range changes {
		switch c.Severity {
		case "high":
			report.HighSeverity++
		case "medium":
			report.MediumSeverity++
		case "low":
			report.LowSeverity++
		}
	}

	// Generate summary
	if len(changes) == 0 {
		report.Summary = fmt.Sprintf("No breaking changes detected between %s and %s across %d releases.",
			a.fromVersion, a.toVersion, len(a.releases))
	} else {
		report.Summary = fmt.Sprintf("Found %d breaking changes between %s and %s across %d releases. "+
			"Severity breakdown: %d high, %d medium, %d low.",
			len(changes), a.fromVersion, a.toVersion, len(a.releases),
			report.HighSeverity, report.MediumSeverity, report.LowSeverity)
	}

	return report
}

var migrationGuidePrompt = `You are an expert technical writer. Generate a comprehensive migration guide based on the following breaking changes.

The guide should:
1. Start with an overview of the migration scope
2. List changes in order of priority (high severity first)
3. Provide code examples where helpful
4. Include a checklist of required changes
5. Note any dependencies that need updating

Format the guide in clear, readable markdown.

Breaking changes to address:
`

// generateMigrationGuide creates a migration guide for the breaking changes
func (a *BreakingChangeAgent) generateMigrationGuide(changes []BreakingChange) (string, error) {
	if len(changes) == 0 {
		return "", nil
	}

	changesJSON, err := json.MarshalIndent(changes, "", "  ")
	if err != nil {
		return "", err
	}

	prompt := migrationGuidePrompt + "\n" + string(changesJSON)
	systemPrompt := fmt.Sprintf("You are a technical documentation assistant helping users migrate from %s to %s of %s.",
		a.fromVersion, a.toVersion, a.repo)

	resp, err := a.bedrock.Chat(prompt, systemPrompt)
	if err != nil {
		return "", err
	}

	if len(resp) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return resp[0].Text, nil
}

// GetState returns the current agent state
func (a *BreakingChangeAgent) GetState() AgentState {
	return a.state
}

// GetReport returns the generated report
func (a *BreakingChangeAgent) GetReport() *BreakingChangeReport {
	return a.report
}

// FormatReport formats the report for display
func FormatReport(report *BreakingChangeReport) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString("                  BREAKING CHANGE REPORT                        \n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Repository:     %s\n", report.Repository))
	sb.WriteString(fmt.Sprintf("Version Range:  %s → %s\n", report.FromVersion, report.ToVersion))
	sb.WriteString(fmt.Sprintf("Releases:       %d analyzed\n", report.TotalReleases))
	sb.WriteString(fmt.Sprintf("Breaking:       %d changes found\n\n", report.BreakingCount))

	if report.BreakingCount > 0 {
		sb.WriteString("SEVERITY BREAKDOWN:\n")
		sb.WriteString(fmt.Sprintf("  🔴 High:   %d\n", report.HighSeverity))
		sb.WriteString(fmt.Sprintf("  🟡 Medium: %d\n", report.MediumSeverity))
		sb.WriteString(fmt.Sprintf("  🟢 Low:    %d\n\n", report.LowSeverity))

		sb.WriteString("───────────────────────────────────────────────────────────────\n")
		sb.WriteString("DETAILED CHANGES:\n")
		sb.WriteString("───────────────────────────────────────────────────────────────\n\n")

		// Group by severity
		for _, severity := range []string{"high", "medium", "low"} {
			for _, c := range report.Changes {
				if c.Severity != severity {
					continue
				}

				icon := "🟢"
				if c.Severity == "high" {
					icon = "🔴"
				} else if c.Severity == "medium" {
					icon = "🟡"
				}

				sb.WriteString(fmt.Sprintf("%s [%s] %s (%s)\n", icon, c.Version, c.Package, c.Type))
				sb.WriteString(fmt.Sprintf("   Description: %s\n", c.Description))
				if c.Migration != "" {
					sb.WriteString(fmt.Sprintf("   Migration:   %s\n", c.Migration))
				}
				sb.WriteString(fmt.Sprintf("   URL:         %s\n\n", c.URL))
			}
		}
	}

	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("SUMMARY:\n")
	sb.WriteString(report.Summary + "\n")
	sb.WriteString("───────────────────────────────────────────────────────────────\n")

	if report.MigrationGuide != "" {
		sb.WriteString("\n")
		sb.WriteString("═══════════════════════════════════════════════════════════════\n")
		sb.WriteString("                     MIGRATION GUIDE                            \n")
		sb.WriteString("═══════════════════════════════════════════════════════════════\n\n")
		sb.WriteString(report.MigrationGuide)
		sb.WriteString("\n")
	}

	return sb.String()
}
