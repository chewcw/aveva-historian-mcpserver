.PHONY: build run test lint clean cross inspector

build:
	go build -o bin/aveva-historian-mcp ./cmd/server

run: build
	./bin/aveva-historian-mcp

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -rf bin/

cross:
	GOOS=windows GOARCH=amd64 go build -o bin/aveva-historian-mcp.exe ./cmd/server

inspector: build
	npx -y @modelcontextprotocol/inspector@latest /home/ccw/Documents/code/rnd/aveva-historian-mcpserver/.worktrees/feat/phase1-foundation/bin/aveva-historian-mcp
