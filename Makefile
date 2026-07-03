.PHONY: test server wasm docker-up docker-down fmt ebitdock ebitdock-down ebitdock-doctor

test:
	go test ./...

fmt:
	gofmt -w cmd internal

server:
	go run ./cmd/server

wasm:
	GOOS=js GOARCH=wasm go build -o web/game.wasm ./cmd/client
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

docker-up:
	docker compose up --build

docker-down:
	docker compose down

ebitdock:
	ebitdock dev

ebitdock-down:
	ebitdock down

ebitdock-doctor:
	ebitdock doctor
