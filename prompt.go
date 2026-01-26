package main

import (
	"bytes"
	"fmt"
	"html/template"
)

// Legacy prompts for keyword search mode (backward compatibility)
var claudeUserPrompt = `Filter the results to only match on packages that contain the keywords {{.Keyword}}. If no matches are found then just say no matches were found. Don't provide inaccurate version information.`

var claudeSystemPrompt = `You are analyzing GitHub release history data for the {{.Repo}} repository.

Extract the package names and change details. The extracted data should include:
- Version number
- Created and published dates
- HTML URL
- Key features or changes for each package

Be concise yet informative, covering major updates without getting bogged down in technical minutiae.
If possible, highlight any trends or significant shifts in development focus over time.

Respond in JSON format with:
{
  "summary": "One or two line summary of the changes",
  "changes": [
    {
      "package": "package name",
      "version": "release version",
      "date": "created/published date",
      "url": "html url",
      "changes": ["list of changes"]
    }
  ]
}

Github release history: {{.History}}`

// Prompt applies data to a template string using Go's template engine
func Prompt[T any](templateString string, data T) string {
	tmpl := template.Must(template.New("").Parse(templateString))

	var buffer bytes.Buffer
	err := tmpl.Execute(&buffer, data)
	if err != nil {
		fmt.Printf("Failed to execute template: %v\n", err)
		return ""
	}

	return buffer.String()
}
