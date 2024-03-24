package main

import "encoding/json"

type Claude struct {
	model   string
	version string
}

func (c Claude) Request(userMsg, systemMsg string) ([]byte, error) {
	req := ClaudeRequest{
		Version:           c.version,
		System:            systemMsg,
		MaxTokensToSample: 500,
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						ContentType: "text",
						Text: Prompt(claudeUserPrompt,
							struct{ Keyword string }{Keyword: userMsg}),
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (c Claude) GetModel() string {
	return c.model
}

func (c Claude) GetModelVersion() string {
	return c.version
}
