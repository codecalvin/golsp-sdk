build:
	go build -o golsp-sdk -mod=vendor ./cmd/golsp-sdk/main.go

test:
	go test -cover -race -mod=vendor ./...

chores:
	go mod tidy
	go mod vendor