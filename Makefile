BUILD_DIR := bin

MAIN_DIR := cmd/awe/*
BINARY_NAME := awe
AWE_INSTALL := github.com/alan-mat/awe/cmd/awe

PROTO_SRC_DIR := proto

GEN_GO_DIR := internal/proto

.DEFAULT_GOAL := build

.PHONY: generate
generate:
	@echo "Generating protobuf code from $(PROTO_SRC_DIR) ..."
	@mkdir -p $(GEN_GO_DIR)
	protoc -I$(PROTO_SRC_DIR) \
		--go_out=paths=source_relative:$(GEN_GO_DIR) \
		--go-grpc_out=paths=source_relative:$(GEN_GO_DIR) \
		$(PROTO_SRC_DIR)/*.proto

.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to GOBIN ..."
	go install $(AWE_INSTALL)

.PHONY: build
build:
	@echo "Creating build dir ($(BUILD_DIR)) ..."
	@mkdir -p $(BUILD_DIR)

	@echo "Building binary under $(BUILD_DIR)/$(BINARY_NAME) ..."
	go build -v -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_DIR)

.PHONY: clean
clean:
	@echo "Cleaning build dir ($(BUILD_DIR)) ..."
	@rm -rf $(BUILD_DIR)
