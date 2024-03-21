package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type Chat interface {
	Request(userMsg, systemMsg string) ([]byte, error)
	GetModel() string
	GetModelVersion() string
}

type Embedding interface {
	Generate() []byte
}

type BedrockService struct {
	client         *bedrockruntime.Client
	chatModel      Chat
	embeddingModel Embedding
}

func NewBedRockService(region string, m Chat) (BedrockService, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return BedrockService{}, fmt.Errorf("failed to load default aws configuration, %v", err)
	}

	return BedrockService{
		client: bedrockruntime.NewFromConfig(cfg),
		chatModel: Claude{
			model:   m.GetModel(),
			version: m.GetModelVersion(),
		},
	}, nil
}

func (b BedrockService) Chat(userMsg, systemMsg string) ([]Content, error) {
	jsonData, err := b.chatModel.Request(userMsg, systemMsg)
	if err != nil {
		return nil, err
	}

	result, err := b.invokeInference(ModelInput{
		Name:        b.chatModel.GetModel(),
		ContentType: "application/json",
		Body:        jsonData,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to invoke inference api, %v", err)
	}

	var response ClaudeResponse
	err = json.Unmarshal(result.Body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response, %v", err)
	}

	return response.Content, nil
}

func (b BedrockService) invokeInference(input ModelInput) (*bedrockruntime.InvokeModelOutput, error) {
	return b.client.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(input.Name),
		ContentType: aws.String(input.ContentType),
		Body:        input.Body,
	})
}
