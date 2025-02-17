package ollama

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type TemplateData struct {
	Content string
}

func RenderPrompt(prompt string, data TemplateData) (string, error) {
	tpl := template.New("prompt").
		Funcs(template.FuncMap{
			"join": strings.Join,
		})

	tpl, err := tpl.Parse(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template, err:%v", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to parse execute template, err:%v", err)
	}

	return buf.String(), nil
}
