package option

type OllamaOption struct {
	Host       string                 `yaml:"Host"`
	Model      string                 `yaml:"Model"`
	Stream     bool                   `yaml:"Stream"`
	Parameters map[string]interface{} `yaml:"Parameters"`
}
