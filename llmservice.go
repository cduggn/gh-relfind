package main

import (
	"log"
	"strings"
)

type LLMService[T any, R any] interface {
	DetectPackageChanges(input T) (R, error)
}

type BedrockInput struct {
	SystemMessage string
	UserMessage   []Message
}

func NewLLMService() (LLMService[BedrockInput, string], error) {
	return AWSBedrockService{}, nil
}

type AWSBedrockService struct {
}

func (a AWSBedrockService) DetectPackageChanges(input BedrockInput) (string, error) {

	bedrock, err := NewBedrockRuntime(awsRegion)
	if err != nil {
		log.Fatalf("Failed to create client: %v ", err)
	}
	rsp, err := bedrock.Inference(modelId, runtimeVersion, input.SystemMessage, maxTokens, input.UserMessage)
	if err != nil {
		log.Fatalf("Failed to get response: %v ", err)
	}

	var sb strings.Builder
	for _, content := range rsp {

		sb.WriteString(content.Text)
	}

	return sb.String(), nil
}
