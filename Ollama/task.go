package Ollama

type ITask interface {
	Do(*OllamaClient)
	BuildRequestPayload(*OllamaClient) *RequestPayload
}
