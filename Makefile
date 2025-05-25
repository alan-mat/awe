BUILD_DIR := bin

MAIN_DIR := cmd/awe/*
BINARY_NAME := awe
AWE_INSTALL := github.com/alan-mat/awe/cmd/awe

AWESOME_MAIN_DIR := cmd/awesome/*
AWESOME_INSTALL := github.com/alan-mat/awe/cmd/awesome
AWESOME_BINARY_NAME := awesome

PROTO_SRC_DIR := proto

GEN_GO_DIR := internal/proto

.PHONY: build run generate install clean

.DEFAULT_GOAL := build

generate:
	@echo "Generating protobuf code from $(PROTO_SRC_DIR) ..."
	@mkdir -p $(GEN_GO_DIR)
	protoc -I$(PROTO_SRC_DIR) \
		--go_out=paths=source_relative:$(GEN_GO_DIR) \
		--go-grpc_out=paths=source_relative:$(GEN_GO_DIR) \
		$(PROTO_SRC_DIR)/*.proto

install: build
	@echo "Installing $(BINARY_NAME) to GOBIN ..."
	go install $(AWE_INSTALL)
	@echo "Installing $(AWESOME_BINARY_NAME) to GOBIN ..."
	go install $(AWESOME_INSTALL)

build:
	@echo "Creating build dir ($(BUILD_DIR)) ..."
	@mkdir -p $(BUILD_DIR)

	@echo "Building binary under $(BUILD_DIR)/$(BINARY_NAME) ..."
	go build -v -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_DIR)

	@echo "Building binary under $(BUILD_DIR)/$(AWESOME_BINARY_NAME) ..."
	go build -v -o $(BUILD_DIR)/$(AWESOME_BINARY_NAME) $(AWESOME_MAIN_DIR)

clean:
	@echo "Cleaning build dir ($(BUILD_DIR)) ..."
	@rm -rf $(BUILD_DIR)
