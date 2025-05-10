MAIN_DIR := .

BUILD_DIR := bin

BINARY_NAME := awe

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

install:
	@echo "Installing $(BINARY_NAME) tp GOBIN ..."
	go install $(MAIN_DIR)

build:
	@echo "Creating build dir ($(BUILD_DIR)) ..."
	@mkdir -p $(BUILD_DIR)

	@echo "Building binary under $(BUILD_DIR)/$(BINARY_NAME) ..."
	go build -v -o $(BUILD_DIR)/$(BINARY_NAME) .

run: build
	@echo "Running $(BINARY_NAME) ..."
	$(BUILD_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning build dir ($(BUILD_DIR)) ..."
	@rm -rf $(BUILD_DIR)
