package config

type Repo struct {
	ID       string `yaml:"id"`
	URL      string `yaml:"url"`
	GPG      string `yaml:"gpg"`
	GPGCheck *int   `yaml:"gpgcheck"`
}

type Opt struct {
	Parent string `yaml:"parent"`
}

type Config struct {
	Repos    []Repo `yaml:"repos"`
	Opts     Opt
	Packages []string `yaml:"packages"`
	Arch     []string `yaml:"arch"`
	Cmds     []string `yaml:"cmds"`
}
