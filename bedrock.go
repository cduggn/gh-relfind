package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"log"
	"log/slog"
	"strings"
)

type Request struct {
	InputText string `json:"inputText"`
}

type Response struct {
	Embedding           []float64 `json:"embedding"`
	InputTextTokenCount int       `json:"inputTextTokenCount"`
}

type BedrockRuntime struct {
	client *bedrockruntime.Client
}

func NewBedrockRuntime(region string) (BedrockRuntime, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return BedrockRuntime{}, fmt.Errorf("failed to load configuration, %v", err)
	}

	return BedrockRuntime{
		client: bedrockruntime.NewFromConfig(cfg),
	}, nil
}

func (b BedrockRuntime) invokeInference(model, contentType string, body []byte) (*bedrockruntime.InvokeModelOutput, error) {
	return b.client.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(model),
		ContentType: aws.String(contentType),
		Body:        body,
	})
}

func (b BedrockRuntime) Chat(modelId, version, systemMsg string, maxTokens int, m []Message) ([]Content, error) {
	request := ClaudeRequest{
		Version:           version,
		MaxTokensToSample: maxTokens,
		System:            systemMsg,
		Messages:          m,
	}

	body, err := json.Marshal(request)
	if err != nil {
		slog.Error("Couldn't marshal the request: ", err)
		return nil, err
	}

	result, err := b.invokeInference(modelId, "application/json", body)

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such host") {
			slog.Error("Error: The Bedrock service is not available in the selected region. Please double-check the service availability for your region at https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/.\n")
		} else if strings.Contains(errMsg, "Could not resolve the foundation model") {
			slog.Error("Error: Could not resolve the foundation model from model identifier: \"%v\". Please verify that the requested model exists and is accessible within the specified region.\n", modelId)
		} else {
			slog.Error("Error: Couldn't invoke Anthropic Claude. Here's why: %v\n", err)
		}
		return nil, err
	}

	var response ClaudeResponse
	err = json.Unmarshal(result.Body, &response)
	if err != nil {
		slog.Error("failed to unmarshal", err)
		return nil, err
	}

	return response.Content, nil
}

func (b BedrockRuntime) Embeddings(modelI string) {
	payload := Request{
		InputText: "this is a test message",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	output, err := b.invokeInference("amazon.titan-embed-text-v1", "application/json", payloadBytes)

	if err != nil {
		log.Fatal("failed to invoke model: ", err)
	}

	var resp Response

	err = json.Unmarshal(output.Body, &resp)

	if err != nil {
		log.Fatal("failed to unmarshal", err)
	}

	fmt.Println("embedding vector from LLM\n", resp.Embedding)
	fmt.Println()

	fmt.Println("generated embedding for input -", "this is a test message")
	fmt.Println("generated vector length -", len(resp.Embedding))

}
