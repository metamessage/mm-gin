.PHONY: help build test clean push push-force

# 默認目標
.DEFAULT_GOAL := help

# 變量
BINARY_NAME := mm-gin
GO := go
GIT := git
REMOTE := origin
BRANCH := main

## help: 顯示幫助信息
help:
	@echo "使用方法:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			target = $$1; sub(/:$$/, "", target); \
			printf "  \033[36m%-15s\033[0m %s\n", target, substr(lastLine, RSTART + 3, RLENGTH); \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

## build: 編譯項目
build:
	$(GO) build -v ./...

## test: 運行測試
test:
	$(GO) test -v ./...

## tidy: 整理依賴
tidy:
	$(GO) mod tidy

## clean: 清理構建產物
clean:
	$(GO) clean
	rm -f $(BINARY_NAME)

## fmt: 格式化代碼
fmt:
	$(GO) fmt ./...

## vet: 靜態檢查
vet:
	$(GO) vet ./...

## lint: 運行所有檢查
lint: fmt vet

## git-init: 初始化 Git 倉庫
git-init:
	$(GIT) init
	$(GIT) config user.email "dev@metamessage.github.io"
	$(GIT) config user.name "MetaMessage Dev"

## git-add: 添加所有文件到 Git
git-add:
	$(GIT) add -A

## commit: 提交更改 (使用默認消息)
commit: git-add
	$(GIT) commit -m "Update: $$(date '+%Y-%m-%d %H:%M:%S')" || echo "No changes to commit"

## commit-msg: 提交更改 (使用自定義消息)
# 用法: make commit-msg MSG="你的提交消息"
commit-msg: git-add
	@if [ -z "$(MSG)" ]; then \
		echo "錯誤: 請提供提交消息，例如: make commit-msg MSG='修復 bug'"; \
		exit 1; \
	fi
	$(GIT) commit -m "$(MSG)"

## remote-add: 添加遠程倉庫
remote-add:
	$(GIT) remote add $(REMOTE) git@github.com:metamessage/mm-gin.git 2>/dev/null || \
	$(GIT) remote set-url $(REMOTE) git@github.com:metamessage/mm-gin.git

## remote-add-https: 添加遠程倉庫 (HTTPS 方式)
# 用法: make remote-add-https TOKEN=your_github_token
remote-add-https:
	@if [ -z "$(TOKEN)" ]; then \
		echo "錯誤: 請提供 GitHub Token，例如: make remote-add-https TOKEN=ghp_xxx"; \
		exit 1; \
	fi
	$(GIT) remote add $(REMOTE) https://$(TOKEN)@github.com/metamessage/mm-gin.git 2>/dev/null || \
	$(GIT) remote set-url $(REMOTE) https://$(TOKEN)@github.com/metamessage/mm-gin.git

## push: 推送到遠程倉庫 (main 分支)
push: remote-add
	$(GIT) branch -M $(BRANCH)
	$(GIT) push -u $(REMOTE) $(BRANCH)

## push-force: 強制推送到遠程倉庫 (謹慎使用)
push-force: remote-add
	$(GIT) branch -M $(BRANCH)
	$(GIT) push -u $(REMOTE) $(BRANCH) --force

## push-all: 完整推送流程 (添加、提交、推送)
push-all: commit push
	@echo "✅ 代碼已成功推送到 GitHub!"

## push-all-msg: 完整推送流程 (帶自定義消息)
# 用法: make push-all-msg MSG="你的提交消息"
push-all-msg: commit-msg push
	@echo "✅ 代碼已成功推送到 GitHub!"

## status: 查看 Git 狀態
status:
	$(GIT) status

## log: 查看提交歷史
log:
	$(GIT) log --oneline --graph -10

## pull: 拉取遠程更新
pull:
	$(GIT) pull $(REMOTE) $(BRANCH)

## all: 完整構建流程 (整理依賴、格式化、構建、測試)
all: tidy lint build test
	@echo "✅ 構建完成!"
