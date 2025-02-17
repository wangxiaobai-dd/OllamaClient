package ollama

type RequestPayload struct {
	Model  string      `json:"model"`
	Prompt string      `json:"prompt"`
	Stream bool        `json:"stream"`
	Format *FormatSpec `json:"format,omitempty"`
}

type FormatSpec struct {
	Type       string                 `json:"type"`
	Properties map[string]FormatField `json:"properties"`
	Required   []string               `json:"required"`
}

type FormatField struct {
	Type string `json:"type"`
}

type ApiResponse struct {
	Response string `json:"response"` // 可能需要二次解析
	Done     bool   `json:"done"`
}

type CodeCheckResult struct {
	File string `json:"file"`
	Line string `json:"line"`
	Bug  string `json:"bug"`
}
