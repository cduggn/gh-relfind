package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"log/slog"
	"strings"
)

type Chat interface {
	Request(userMsg, systemMsg string) ([]byte, error)
}

type Embedding interface {
	Generate() []byte
}

type BedrockService struct {
	client         *bedrockruntime.Client
	chatModel      Chat
	embeddingModel Embedding
}

func NewBedRockService(region string) (BedrockService, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return BedrockService{}, fmt.Errorf("failed to load configuration, %v", err)
	}

	return BedrockService{
		client:    bedrockruntime.NewFromConfig(cfg),
		chatModel: Chat(Claude{}),
	}, nil
}

func (b BedrockService) SetChatModel(model Chat) {
	b.chatModel = model
}

func (b BedrockService) SetEmbeddingModel(model Embedding) {
	b.embeddingModel = model
}

func (b BedrockService) Chat(userMsg, systemMsg string) (string, error) {
	jsonData, err := b.chatModel.Request(userMsg, systemMsg)
	if err != nil {
		return "", err
	}

	result, err := b.invokeInference(ModelInput{
		Name:        "anthropic.claude-3-sonnet-20240229-v1:0",
		ContentType: "application/json",
		Body:        jsonData,
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such host") {
			slog.Error("Error: The Bedrock service is not available in the selected region. Please double-check the service availability for your region at https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/.\n")
		} else if strings.Contains(errMsg, "Could not resolve the foundation model") {
			slog.Error("Error: Could not resolve the foundation model from model identifier: \"%v\". Please verify that the requested model exists and is accessible within the specified region.\n", modelId)
		} else {
			slog.Error("Error: Couldn't invoke Anthropic Claude. Here's why: %v\n", err)
		}
		return "", err
	}

	var response ClaudeResponse
	err = json.Unmarshal(result.Body, &response)
	if err != nil {
		slog.Error("failed to unmarshal", err)
		return "", err
	}

	for _, message := range response.Content {
		fmt.Println(message.Text)
	}

	return "fmt.Sprintf(response.Content)", nil
}

//func (b BedrockService) GenerateEmbeddings(in string) (string, error) {
//	payload, err := json.Marshal(in)
//	if err != nil {
//		log.Fatal(err)
//	}
//	output, err := b.invokeInference(ModelInput{
//		Name:        "amazon.titan-embed-text-v1",
//		ContentType: "application/json",
//		Body:        payload,
//	})
//	if err != nil {
//		log.Fatal("failed to invoke model: ", err)
//	}
//
//	var resp Response
//	err = json.Unmarshal(output.Body, &resp)
//	if err != nil {
//		log.Fatal("failed to unmarshal", err)
//	}
//
//	fmt.Println("embedding vector from LLM\n", resp.Embedding)
//	fmt.Println("generated embedding for input -", "this is a test message")
//	fmt.Println("generated vector length -", len(resp.Embedding))
//
//	return "Embeddings generated", nil
//}

func (b BedrockService) invokeInference(input ModelInput) (*bedrockruntime.InvokeModelOutput, error) {
	return b.client.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(input.Name),
		ContentType: aws.String(input.ContentType),
		Body:        input.Body,
	})
}
