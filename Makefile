test:
	go test ./...

run:
	go run ./cmd/trader

fmt:
	gofmt -w cmd internal
