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
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/alan-mat/awe/server"
	"github.com/alan-mat/awe/worker"
	"github.com/alexflint/go-arg"
)

const (
	ProgramName   = "AWE"
	Version       = "v0.0.1"
	RepositoryUrl = "github.com/alan-mat/awe"
)

type serveCmd struct{}

type workerCmd struct{}

/* type workflowCmd struct {
	List    *workflowListCmd `arg:"subcommand" help:"list available workflows"`
	Execute *workflowExecCmd `arg:"subcommand:exec" help:"execute a workflow on the server"`
	Host    string           `arg:"--host,-H" default:"localhost" help:"server host address"`
	Port    uint             `arg:"--port,-p" default:"50051" help:"server port"`
}

type workflowListCmd struct{}

type workflowExecCmd struct{} */

type args struct {
	Server *serveCmd  `arg:"subcommand:serve" help:"start the AWE server"`
	Worker *workerCmd `arg:"subcommand:work" help:"start the AWE worker"`
	// Workflow *workflowCmd `arg:"subcommand" help:"manage workflows"`

	ConfigPath string `arg:"-c,--config" default:"awe-config.yaml" help:"path to the config file" placeholder:""`
}

func (args) Version() string {
	return fmt.Sprintf("%s %s", ProgramName, Version)
}

func (args) Epilogue() string {
	return fmt.Sprintf("For more information visit %s", RepositoryUrl)
}

func main() {
	var args args

	p, err := arg.NewParser(arg.Config{Program: strings.ToLower(ProgramName)}, &args)
	if err != nil {
		log.Fatalf("there was an error in the definition of the Go struct: %v", err)
	}
	p.MustParse(os.Args[1:])

	if p.Subcommand() == nil {
		p.WriteUsage(os.Stdout)
		os.Exit(0)
	}

	conf, err := ReadConfig(args.ConfigPath)
	if err != nil {
		fmt.Println(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	var cmd func(any, *config) error

	switch p.Subcommand().(type) {
	case *serveCmd:
		cmd = startServer
	case *workerCmd:
		cmd = startWorker
	default:
		p.FailSubcommand("unrecognized command", p.SubcommandNames()...)
	}

	cmd(p.Subcommand(), conf)
}

func startServer(args any, conf *config) error {
	var serverConfig server.ServerConfig
	if conf == nil {
		serverConfig = server.DefaultConfig()
	} else {
		serverConfig = server.ServerConfig{
			ListenHost:    conf.Server.ListenHost,
			ListenPort:    conf.Server.ListenPort,
			RedisAddr:     conf.Transport.Addr,
			RedisUsername: conf.Transport.Username,
			RedisPassword: conf.Transport.Password,
			RedisDB:       conf.Transport.DB,
		}
	}

	srv := server.New(serverConfig)
	return srv.Serve()
}

func startWorker(args any, conf *config) error {
	var workerConfig worker.WorkerConfig
	var workflows string
	if conf == nil {
		workerConfig = worker.DefaultConfig()
	} else {
		workerConfig = worker.WorkerConfig{
			Workers:       conf.Worker.Workers,
			RedisAddr:     conf.Transport.Addr,
			RedisUsername: conf.Transport.Username,
			RedisPassword: conf.Transport.Password,
			RedisDB:       conf.Transport.DB,
			QdrantHost:    conf.VectorStore.Host,
			QdrantPort:    conf.VectorStore.Port,
		}
		workflows = conf.WorkflowConfigPath
	}

	worker := worker.New(workerConfig)
	err := worker.RegisterWorkflows(workflows)
	if err != nil {
		return err
	}

	return worker.Start()
}
