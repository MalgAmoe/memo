package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// CallLLM sends a prompt to the LLM and returns the response
func CallLLM(prompt string) (string, error) {
	llmURL := "https://api.fireworks.ai/inference/v1/chat/completions"
	llmModel := "accounts/fireworks/models/kimi-k2p5"
	llmKey := os.Getenv("FIREWORKS_API_KEY")

	if u := os.Getenv("LLM_URL"); u != "" {
		llmURL = u
	}
	if m := os.Getenv("LLM_MODEL"); m != "" {
		llmModel = m
	}

	if llmKey == "" {
		return "", fmt.Errorf("FIREWORKS_API_KEY not set")
	}

	reqBody := chatRequest{
		Model: llmModel,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", llmURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llmKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM service error: %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return result.Choices[0].Message.Content, nil
}

// GenerateBrief synthesizes a project brief from memories
func GenerateBrief(projectName string, currentBrief string, memories []Memory) (string, error) {
	var memList string
	for _, m := range memories {
		memList += fmt.Sprintf("- [%s] %s\n", m.Type, m.Content)
	}

	var prompt string
	if currentBrief == "" {
		prompt = fmt.Sprintf(`You are synthesizing a project brief from individual memory fragments.

Project: %s

Memories:
%s

Write a concise project brief (3-5 paragraphs) that synthesizes these fragments into a coherent understanding. Cover:
- What the project is and its purpose
- Key technical decisions and why they were made
- Current state and recent developments
- Important gotchas or things to remember

Write in present tense, as a reference document. No headers, no bullet points — flowing prose that gives someone complete context to work on this project. Be specific, not generic.`, projectName, memList)
	} else {
		prompt = fmt.Sprintf(`You are updating a project brief with new information.

Project: %s

Current brief:
%s

All memories (including new ones):
%s

Update the brief to incorporate any new information. Keep it 3-5 paragraphs of flowing prose. Preserve important existing context. If new memories contradict old information, favor the new. No headers, no bullet points.`, projectName, currentBrief, memList)
	}

	return CallLLM(prompt)
}
