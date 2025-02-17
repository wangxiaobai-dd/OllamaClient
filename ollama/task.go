package ollama

type ITask interface {
	Do(*OllamaClient)
	BuildRequestPayload(*OllamaClient) *RequestPayload
}
