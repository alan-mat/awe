# Agentic Workflow Engine

AWE is an API server for hosting and executing **agentic workflows**, written in Golang. It is based on the [Modular RAG](https://arxiv.org/abs/2407.21059v1) framework. It can be used for simple RAG pipelines, complex AI Agents or any other LLM-based application. 

> [!WARNING]
> AWE is currently being developed and finds itself in a very early
> alpha stage. There are no client libraries, APIs and configurations will change, things will break. 
> It is not yet recommended for use in production. 
> If you know what you're doing, feel free to experiment with it
> and share your feedback. It will greatly further AWE's development.

## Features

- Define workflows using YAML
- Durable request execution
- Streaming responses
- Multiple providers (e.g. OpenAI, Mistral, Ollama, etc.)
- gRPC API
- Distributed architecture

## Prerequisites

- Go 1.21 or higher

## Installation

```bash
go install github.com/alan-mat/awe/cmd/awe
```

or

```bash
git clone https://github.com/alan-mat/awe.git
cd awe
go mod download
make install
```

As of now, AWE requires a running Redis and Qdrant instance. The fastest way to get started is by using docker:

```bash
# Redis
docker run -d --name redis -p 6379:6379 redis:latest

# Qdrant
docker run -d --name qdrant -p 6333:6333 -p 6334:6334 \
    -v "$(pwd)/qdrant_storage:/qdrant/storage:z" \
    qdrant/qdrant
```

## Configuration

Create a `.env` file in the root directory to set provider API keys and export them:

```env
OPENAI_API_KEY=
GEMINI_API_KEY=
MISTRAL_API_KEY=
JINA_API_KEY=
COHERE_API_KEY=
TAVILY_API_KEY=
```

The default app configuration can be found under `configs/default-awe-config.yaml`. To override it it's recommended to copy it and set your own values.

## Defining workflows

Coming soon...

## Usage

First, make sure your Redis and Qdrant instances are running.

Start the server:

```bash
awe serve -c <path-to-config>
```

Start the worker:

```bash
awe work -c <path-to-config>
```

The gRPC server will start on `localhost:50051` by default.

You can start testing it out using any gRPC client, as long as you provide the `.proto` files. For example using [grpc-client-cli](https://github.com/vadimi/grpc-client-cli):

```bash
grpc-client-cli --proto ./proto/awe.proto :50051
```

## API 

The API is defined using Protobuf. 

As of now there is no client library. You can copy the `/proto` directory manually or vendor it, generate source files using `protoc` (view [Makefile](Makefile)) and use the gRPC client directly. 

## Roadmap

Coming soon...

## Contributing

Any contributions are welcome. Feel free to fork this repository and then open a Pull Request :)

## License

This project is licensed under the MIT [License](LICENSE).
