.PHONY: build test clean run help

# 变量
BINARY_NAME = gclaw
GO = go
GOFLAGS = -v

# 默认目标
all: build

# 构建
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build -o $(BINARY_NAME) ./cmd/gclaw
	@echo "Build complete: $(BINARY_NAME)"

# 构建发布版本
build-release:
	@echo "Building release version..."
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/gclaw

# 运行测试
test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

# 运行测试并显示覆盖率
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test $(GOFLAGS) -cover ./...

# 清理
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf sessions/
	$(GO) clean
	@echo "Clean complete"

# 运行
run: build
	./$(BINARY_NAME) summary

# 格式化代码
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# 安装
install: build
	@echo "Installing..."
	cp $(BINARY_NAME) $(GOPATH)/bin/ 2>/dev/null || echo "Note: GOPATH not set, binary remains in current directory"

# 帮助
help:
	@echo "gClawCoder Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  build          - Build the binary"
	@echo "  build-release  - Build release binary (optimized)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run summary"
	@echo "  fmt            - Format code"
	@echo "  install        - Install binary to GOPATH/bin"
	@echo "  help           - Show this help"
