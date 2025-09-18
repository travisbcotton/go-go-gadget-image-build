package config

import (
	"os"
	"fmt"
	"errors"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	c, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expanded := os.Expand(string(c), func(key string) string {
		return os.Getenv(key)
	})

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	applyDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyDefaults(c *Config) {
	if c.Arch == nil {
		c.Arch = []string{"x86_64", "noarch"}
	}
}

func validate(c *Config) error {
	if len(c.Repos) == 0 {
		return errors.New("at least one repo is required")
	}
	for _, r := range c.Repos {
		if r.ID == "" || r.URL == "" {
			return errors.New("each repo needs id and url")
		}
	}
	if len(c.Packages) == 0 {
		return errors.New("packages list cannot be empty")
	}
	return nil
}

