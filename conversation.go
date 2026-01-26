package main

import (
	"encoding/json"
	"fmt"
)

// Conversation manages multi-turn chat history
type Conversation struct {
	Messages   []Message
	Repository string
	Owner      string
	Releases   []Release
}

// NewConversation creates a new conversation context
func NewConversation(owner, repo string) *Conversation {
	return &Conversation{
		Messages:   make([]Message, 0),
		Repository: repo,
		Owner:      owner,
	}
}

// AddUserMessage adds a user message to the conversation
func (c *Conversation) AddUserMessage(text string) {
	c.Messages = append(c.Messages, Message{
		Role: "user",
		Content: []Content{
			{ContentType: "text", Text: text},
		},
	})
}

// AddAssistantMessage adds an assistant message to the conversation
func (c *Conversation) AddAssistantMessage(text string) {
	c.Messages = append(c.Messages, Message{
		Role: "assistant",
		Content: []Content{
			{ContentType: "text", Text: text},
		},
	})
}

// SetReleases caches the fetched releases for the conversation
func (c *Conversation) SetReleases(releases []Release) {
	c.Releases = releases
}

// GetMessages returns all messages in the conversation
func (c *Conversation) GetMessages() []Message {
	return c.Messages
}

// Clear resets the conversation history
func (c *Conversation) Clear() {
	c.Messages = make([]Message, 0)
}

// ConversationRequest extends ClaudeRequest for multi-turn conversations
type ConversationRequest struct {
	Version           string    `json:"anthropic_version"`
	Messages          []Message `json:"messages"`
	MaxTokensToSample int       `json:"max_tokens"`
	System            string    `json:"system"`
}

// BuildConversationRequest creates a request with full conversation history
func BuildConversationRequest(conv *Conversation, systemPrompt string, version string, maxTokens int) ([]byte, error) {
	req := ConversationRequest{
		Version:           version,
		System:            systemPrompt,
		MaxTokensToSample: maxTokens,
		Messages:          conv.GetMessages(),
	}

	return json.Marshal(req)
}

// ConversationService handles multi-turn conversations with Claude
type ConversationService struct {
	bedrock      BedrockService
	conversation *Conversation
	systemPrompt string
}

// NewConversationService creates a new conversation service
func NewConversationService(bedrock BedrockService, owner, repo string) *ConversationService {
	return &ConversationService{
		bedrock:      bedrock,
		conversation: NewConversation(owner, repo),
	}
}

// SetSystemPrompt sets the system prompt for the conversation
func (cs *ConversationService) SetSystemPrompt(prompt string) {
	cs.systemPrompt = prompt
}

// SetReleases caches releases in the conversation context
func (cs *ConversationService) SetReleases(releases []Release) {
	cs.conversation.SetReleases(releases)
}

// GetReleases returns cached releases
func (cs *ConversationService) GetReleases() []Release {
	return cs.conversation.Releases
}

// Chat sends a message and gets a response, maintaining conversation history
func (cs *ConversationService) Chat(userMessage string) (string, error) {
	cs.conversation.AddUserMessage(userMessage)

	jsonData, err := BuildConversationRequest(
		cs.conversation,
		cs.systemPrompt,
		runtimeVersion,
		maxTokens,
	)
	if err != nil {
		return "", fmt.Errorf("failed to build conversation request: %w", err)
	}

	result, err := cs.bedrock.invokeInference(ModelInput{
		Name:        cs.bedrock.chatModel.GetModel(),
		ContentType: "application/json",
		Body:        jsonData,
	})
	if err != nil {
		return "", fmt.Errorf("failed to invoke inference: %w", err)
	}

	var response ClaudeResponse
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	assistantResponse := response.Content[0].Text
	cs.conversation.AddAssistantMessage(assistantResponse)

	return assistantResponse, nil
}

// GetConversation returns the underlying conversation
func (cs *ConversationService) GetConversation() *Conversation {
	return cs.conversation
}

// ClearHistory clears the conversation history
func (cs *ConversationService) ClearHistory() {
	cs.conversation.Clear()
}
