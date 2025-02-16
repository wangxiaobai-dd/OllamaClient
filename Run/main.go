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
	option, err := Option.LoadOption("config/option.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(option.Ollama, option.CodeCheck)

	client := Ollama.NewOllamaClient(option.Ollama)
	task := Task.NewCodeCheckTask(option.CodeCheck)
	client.Run(task)
}
