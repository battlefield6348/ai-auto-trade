BINARY := bin/ai-auto-trade
# 常用 Go 指令：格式化、靜態檢查、測試、編譯與執行
MAIN := ./cmd/api
PKG := ./...

.PHONY: fmt fmt-check vet test build run tidy lint ci clean

# 將所有檔案套用 gofmt
fmt:
	gofmt -w .

# 只檢查是否符合 gofmt，若未通過會列出檔案並回傳非 0
fmt-check:
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$files"; \
		exit 1; \
	fi

# Go 官方靜態檢查
vet:
	go vet $(PKG)

# 執行所有測試
test:
	go test $(PKG)

# 編譯可執行檔到 bin/ai-auto-trade
build:
	mkdir -p $(dir $(BINARY))
	go build -o $(BINARY) $(MAIN)

# 直接執行主程式
run:
	go run $(MAIN)

# 整理模組依賴
tidy:
	go mod tidy

# 快速本地檢查：格式 + 靜態檢查
lint: fmt-check vet

# CI 一次跑完格式檢查、靜態檢查與測試
ci: fmt-check vet test

# 移除編譯輸出
clean:
	rm -rf $(BINARY)
