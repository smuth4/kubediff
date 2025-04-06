package config

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	WatchMode RunMode = "watch"
	DiffMode  RunMode = "diff"
)

type RunMode string

type Config struct {
	Mode       RunMode
	Resources  []Resource
	Namespaces []string
	Notifier   Notifier
	IgnoreDiff []string `:yaml:"ignoreDiff"`
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
	return nil
}

func (c *Config) IsIgnoredDiffPath(kind string, path string) bool {
	for _, p := range c.IgnoreDiff {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	for _, r := range c.Resources {
		if r.Kind != kind {
			continue
		}
		for _, p := range r.IgnoreDiff {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
	}
	return false
}

type Resource struct {
	Kind       string
	IgnoreDiff []string `:yaml:"ignoreDiff"`
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

	b, err := ioutil.ReadAll(config)
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
