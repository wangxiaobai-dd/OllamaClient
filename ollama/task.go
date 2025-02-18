package ollama

type ITask interface {
	Do(*OllamaClient)
}
