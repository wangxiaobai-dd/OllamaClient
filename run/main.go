package main

import (
	"OllamaChat/Ollama"
	"OllamaChat/Option"
	"OllamaChat/Task"
	"log"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	option, err := option.LoadOption("config/option.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(option.Ollama, option.CodeCheck)

	client := ollama.NewOllamaClient(option.Ollama)
	task := task.NewCodeCheckTask(option.CodeCheck)
	client.Run(task)
}
