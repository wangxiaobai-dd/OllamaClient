package Option

type CodeCheckOption struct {
	CheckDay    int                    `yaml:"CheckDay"`
	UserName    string                 `yaml:"UserName"`
	Password    string                 `yaml:"Password"`
	ProjectPath string                 `yaml:"ProjectPath"`
	RepoURL     string                 `yaml:"RepoURL"`
	DiffDir     string                 `yaml:"DiffDir"`
	Prompt      string                 `yaml:"Prompt"`
	Format      map[string]interface{} `yaml:"CodeCheckFormat"`
}
