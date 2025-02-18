package ollama

type GeneratePayload struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
	Format  *Format                `json:"format,omitempty"`
}

type Format struct {
	Type       string                 `json:"type"`
	Properties map[string]FormatField `json:"properties"`
	Required   []string               `json:"required"`
}

type FormatField struct {
	Type string `json:"type"`
}

type GenerateResponse struct {
	Response string `json:"response"` // 可能需要二次解析
	Done     bool   `json:"done"`
}
