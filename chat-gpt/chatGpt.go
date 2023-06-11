package chatgpt

import (
	"context"

	"github.com/franciscoescher/goopenai"
)

type ChatGpt struct {
	Client *goopenai.Client
}

func NewClient(organization string, apiKey string) *ChatGpt {
	svc := &ChatGpt{
		Client: goopenai.NewClient(apiKey, organization),
	}
	return svc
}

func (c ChatGpt) ChatCompletion(message string) (string, error) {
	r := goopenai.CreateCompletionsRequest{
		Model: "gpt-4",
		Messages: []goopenai.Message{
			{
				Role:    "user",
				Content: message,
			},
		},
		Temperature: 0.7,
	}
	completions, err := c.Client.CreateCompletions(context.Background(), r)
	if err != nil {
		return "", err
	}
	return completions.Choices[0].Message.Content, nil
}
