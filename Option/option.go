package Option

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Option struct {
	Ollama    *OllamaOption    `yaml:"Ollama"`
	CodeCheck *CodeCheckOption `yaml:"CodeCheck"`
}

func LoadOption(filePath string) (*Option, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var option Option
	err = yaml.Unmarshal(fileContent, &option)
	if err != nil {
		return nil, err
	}
	return &option, nil
}
