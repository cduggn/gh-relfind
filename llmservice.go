package main

import (
	"log"
	"strings"
)

type LLMService[T any, R any] interface {
	DetectPackageChanges(input T) (R, error)
	GenerateEmbeddings(input T) (R, error)
}

type BedrockInput struct {
	SystemMessage string
	UserMessage   []Message
}

func NewLLMService(region string) (LLMService[BedrockInput, string], error) {
	runtime, err := NewBedrockRuntime(region)
	if err != nil {
		return nil, err
	}

	return AWSBedrockService{
		Runtime: runtime,
	}, nil
}

type AWSBedrockService struct {
	Runtime BedrockRuntime
}

func (a AWSBedrockService) DetectPackageChanges(input BedrockInput) (string, error) {
	rsp, err := a.Runtime.Chat(modelId, runtimeVersion, input.SystemMessage, maxTokens, input.UserMessage)
	if err != nil {
		log.Fatalf("Failed to get response: %v ", err)
	}

	var sb strings.Builder
	for _, content := range rsp {

		sb.WriteString(content.Text)
	}

	return sb.String(), nil
}

func (a AWSBedrockService) GenerateEmbeddings(input BedrockInput) (string, error) {
	a.Runtime.Embeddings("amazon.titan-embed-text-v1")

	return "Embeddings generated", nil
}
