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
	Version       = "v0.0.0"
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	var cmd func(any) error

	switch p.Subcommand().(type) {
	case *serveCmd:
		cmd = startServer
	case *workerCmd:
		cmd = startWorker
	default:
		p.FailSubcommand("unrecognized command", p.SubcommandNames()...)
	}

	cmd(p.Subcommand())
}

func startServer(args any) error {
	srv := server.New()
	return srv.Serve()
}

func startWorker(args any) error {
	worker := worker.New()
	err := worker.RegisterWorkflows("workflows.yaml")
	if err != nil {
		return err
	}
	return worker.Start()
}
