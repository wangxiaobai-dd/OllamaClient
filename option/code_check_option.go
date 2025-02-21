package option

type CodeCheckOption struct {
	CheckDay    int    `yaml:"CheckDay"`
	UserName    string `yaml:"UserName"`
	Password    string `yaml:"Password"`
	ProjectPath string `yaml:"ProjectPath"`
	RepoURL     string `yaml:"RepoURL"`
	DiffDir     string `yaml:"DiffDir"`
	OutDir      string `yaml:"OutDir"`
	OutPrefix   string `yaml:"OutPrefix"`
	Prompt      string `yaml:"Prompt"`
	CronTime    string `yaml:"CronTime"`
	UploadURL   string `yaml:"UploadURL"`
}
