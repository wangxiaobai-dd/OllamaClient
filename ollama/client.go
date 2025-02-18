package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"OllamaChat/Option"
)

type OllamaClient struct {
	option *option.OllamaOption
	client *http.Client
}

func NewOllamaClient(option *option.OllamaOption) *OllamaClient {
	client := &http.Client{
		Timeout: time.Second * 300,
	}
	ollamaClient := &OllamaClient{client: client, option: option}
	return ollamaClient
}

func (oc *OllamaClient) GetGeneratePayload() *GeneratePayload {
	payload := &GeneratePayload{
		Model:   oc.option.Model,
		Stream:  oc.option.Stream,
		Options: oc.option.Parameters,
	}
	return payload
}

var TestNum int

func (oc *OllamaClient) Generate(payload *GeneratePayload) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload to json, err:%v\n", err)
	}

	TestNum++
	f, err := os.Create(fmt.Sprintf("%d-payload.json", TestNum))
	if _, err := f.Write(payloadBytes); err != nil {
		return "", fmt.Errorf("failed to write payload to file, err:%v", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/generate", oc.option.Host),
		bytes.NewBuffer(payloadBytes),
	)

	if err != nil {
		return "", fmt.Errorf("failed to create generate request, err:%v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := oc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send generate request, err:%v\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return string(respBody), fmt.Errorf("failed to get ok status when generating, err:%v, resp:%s\n", err, string(respBody))
	}
	var apiResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response, err:%v\n", err)
	}
	return apiResp.Response, nil
}

func (oc *OllamaClient) Run(task ITask) {
	task.Do(oc)
}
