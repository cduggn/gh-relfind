package main

import (
	"bytes"
	"html/template"
)

var claudeUserPrompt = `Filter the results to only match on packages that contain the keywords {{.Keyword}} . If not matches are found then just say no matches were found, Don't provide inaccurate version information`

var claudeSystemPrompt = `Extract the package names and change details from the GitHub release history data for the {{.Repo}} repository? The extracted data should include the version number, created and published dates, html url and the key feature or changes associated with each changed package. The information should be concise yet informative, covering major updates without getting bogged down in technical minutiae. Additionally, if possible, highlight any trends or significant shifts in development focus over time. My communication style is straightforward and factual, aiming for clarity and efficiency in conveying information

The response should be in JSON format consisting of a summary field and an array of changes. The summary field should include a one or two line summary followed by an array of changes related to the {{.Keyword}} package which contains package name, date created and or date published, release number,  and any changes made to the package. 

Github release history: {{.History}}`

func Prompt[T any](templateString string, data T) string {
	template := template.Must(template.New("prompt").Parse(templateString))

	var buffer bytes.Buffer
	template.Execute(&buffer, data)

	return buffer.String()
}
