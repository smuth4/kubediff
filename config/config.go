package config

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"
)

const (
	WatchMode RunMode = "watch"
	DiffMode  RunMode = "diff"
)

type RunMode string

type Config struct {
	Mode            RunMode
	Resources       []Resource
	Namespaces      []string
	Notifier        Notifier
	IgnoreDiff      []string `yaml:"ignoreDiff"`
	ignoreGlobs     []glob.Glob
	ignoreKindGlobs map[string][]glob.Glob
}

func (c *Config) init() {
	if c.Mode == "" {
		c.Mode = WatchMode
	}
	if len(c.Namespaces) == 0 {
		c.Namespaces = append(c.Namespaces, "all")
	}
}

func (c *Config) validate() error {
	for _, ns := range c.Namespaces {
		if ns == "all" {
			if len(c.Namespaces) > 1 {
				return errors.New("cannot specify a namespace after selecting all")
			}
		}
	}
	for _, i := range c.IgnoreDiff {
		fmt.Println(i)
		glob, err := glob.Compile(i)
		if err != nil {
			return fmt.Errorf("failed to compile glob \"%s\": %w", i, err)

		}
		c.ignoreGlobs = append(c.ignoreGlobs, glob)
	}
	c.ignoreKindGlobs = map[string][]glob.Glob{}
	for _, r := range c.Resources {
		if len(r.IgnoreDiff) != 0 {
			globs := []glob.Glob{}
			for _, i := range r.IgnoreDiff {
				fmt.Println(i)
				fmt.Println(r.Kind)
				glob, err := glob.Compile(i)
				if err != nil {
					return fmt.Errorf("failed to compile glob \"%s\" for kind \"%s\": %w", i, r.Kind, err)
				}
				globs = append(globs, glob)
			}
			c.ignoreKindGlobs[r.Kind] = globs
		}
	}
	return nil
}

func (c *Config) IsIgnoredDiffPath(kind string, path string) bool {
	for _, g := range c.ignoreGlobs {
		if g.Match(path) {
			return true
		}
	}
	globs, ok := c.ignoreKindGlobs[kind]
	if ok {
		for _, g := range globs {
			if g.Match(path) {
				return true
			}
		}
	}
	return false
}

type Resource struct {
	Kind        string
	IgnoreDiff  []string `yaml:"ignoreDiff"`
	ignoreGlobs []glob.Glob
}

type Notifier struct {
	Webhook Webhook
	NoOp    NoOp
}

type Webhook struct {
	Enabled bool
	Url     string
}

type NoOp struct {
	Enabled bool
}

// New returns new Config
func New(filepath string) (*Config, error) {
	c := &Config{}
	config, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer config.Close()

	b, err := io.ReadAll(config)
	if err != nil {
		return nil, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}

	c.init()
	err = c.validate()
	if err != nil {
		return nil, err
	}

	return c, nil
}
