package Ollama

import (
	"OllamaChat/Option"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaClient struct {
	option *Option.OllamaOption
	client *http.Client
}

func NewOllamaClient(option *Option.OllamaOption) *OllamaClient {
	client := &http.Client{
		Timeout: time.Second * 300,
	}
	ollamaClient := &OllamaClient{client: client, option: option}
	return ollamaClient
}

func (oc *OllamaClient) GetRequestPayload() *RequestPayload {
	payload := &RequestPayload{
		Model:  oc.option.Model,
		Stream: oc.option.Stream,
		Format: FormatSpec{
			Properties: make(map[string]FormatField),
			Required:   make([]string, 0),
		},
	}
	return payload
}

func (oc *OllamaClient) Generate(payload *RequestPayload) (<-chan ApiResponse, <-chan error) {
	respChan := make(chan ApiResponse)
	errChan := make(chan error, 1)

	go func() {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal payload to json, err:%v", err)
			return
		}
		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/api/generate", oc.option.Host),
			bytes.NewBuffer(payloadBytes),
		)
		if err != nil {
			errChan <- fmt.Errorf("failed to create generate request, err:%v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := oc.client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to send generate request, err:%v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("failed to get ok status when generating, err:%v, resp:%v", err, respBody)
			return
		}
		var apiResp ApiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			errChan <- fmt.Errorf("failed to decode response, err:%v", err)
		}
		respChan <- apiResp
	}()
	return respChan, errChan
}

func (oc *OllamaClient) Run(task ITask) {
	task.Do(oc)
}
