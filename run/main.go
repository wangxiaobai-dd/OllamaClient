package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"OllamaClient/ollama"
	"OllamaClient/option"
	"OllamaClient/task"
	"github.com/spf13/pflag"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	optionFile := pflag.StringP("option", "o", "config/option.yaml", "Path to the yaml configuration file")
	pflag.Parse()
	opt, err := option.LoadOption(*optionFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(opt.Ollama, opt.CodeCheck)

	client := ollama.NewOllamaClient(opt.Ollama)
	ct := task.NewCodeCheckTask(opt.CodeCheck)
	client.Run(ct)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-c
}
