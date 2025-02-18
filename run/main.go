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
	opt, err := option.LoadOption("config/option.yaml")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(opt.Ollama, opt.CodeCheck)

	client := ollama.NewOllamaClient(opt.Ollama)
	ct := task.NewCodeCheckTask(opt.CodeCheck)
	client.Run(ct)
}
