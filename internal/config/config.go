package config

type Repo struct {
	ID       string `yaml:"id"`
	URL      string `yaml:"url"`
}

type Config struct {
	Repos		[]Repo		`yaml:"repos"`
	Packages	[]string 	`yaml:"packages"`
	Arch		[]string		`yaml:"arch"`
}