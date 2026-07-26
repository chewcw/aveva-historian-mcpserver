.PHONY: run test

build:
	go build ./...

run: build
	./aveva-historian-mcpserver

test: build
	npx -y @modelcontextprotocol/inspector ./aveva-historian-mcpserver
