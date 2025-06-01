// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

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
