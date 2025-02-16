package Ollama

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
		return "", fmt.Errorf("模板解析失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("模板渲染失败: %w", err)
	}

	return buf.String(), nil
}
