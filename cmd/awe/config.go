package main

import (
	"os"

	"github.com/goccy/go-yaml"
)

type redisConfig struct {
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type qdrantConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type workerConfig struct {
	Workers int `yaml:"workers"`
}

type serverConfig struct {
	ListenHost string `yaml:"listen_host"`
	ListenPort int    `yaml:"listen_port"`
}

type config struct {
	Server serverConfig `yaml:"server"`
	Worker workerConfig `yaml:"worker"`

	Transport   redisConfig  `yaml:"transport"`
	VectorStore qdrantConfig `yaml:"vector_store"`

	WorkflowConfigPath string `yaml:"workflows"`
}

func ReadConfig(path string) (*config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf config
	if err := yaml.Unmarshal(file, &conf); err != nil {
		return nil, err
	}

	if conf.WorkflowConfigPath == "" {
		conf.WorkflowConfigPath = "default-workflows.yaml"
	}

	return &conf, nil
}
