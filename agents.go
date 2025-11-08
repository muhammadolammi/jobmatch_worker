package main

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

func GetAgent(apiKey, agentName string) (agent.Agent, error) {
	ctx := context.Background()
	model, err := gemini.NewModel(ctx, "gemini-2.5-pro", &genai.ClientConfig{
		APIKey: apiKey,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create model: %v", err)
	}

	customAgent, err := llmagent.New(llmagent.Config{
		Name:        agentName,
		Model:       model,
		Description: "Analyze Resume",
		Instruction: prompt(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %v", err)
	}

	return customAgent, err
}
