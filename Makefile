.PHONY: all
all: download-modules lint test build

export GOPRIVATE=rd-gitlab.egt-bg.com

.PHONY: download-modules
download-modules:
	if [ ! -d ./vendor ]; then go mod download; fi

.PHONY: tidy-modules
tidy-modules:
	go mod tidy

.PHONY: update-modules
update-modules:
	go mod tidy
	go get -t -u ./...
	go mod tidy

.PHONY: lint
lint:
	if [ ! -d ./vendor ]; then golangci-lint run ./...; else golangci-lint run --modules-download-mode=vendor ./...; fi

.PHONY: lint-fix
lint-fix:
	if [ ! -d ./vendor ]; then golangci-lint run --fix ./...; else golangci-lint run --modules-download-mode=vendor --fix ./...; fi

.PHONY: test
test:
	go test ./... -race -cover -count=1

.PHONY: build
build:
	CGO_ENABLED=1 go build -trimpath -o ./bin/01-first-example-server ./examples/01-first-example/server/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/01-first-example-client ./examples/01-first-example/client/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/02-weather-update-server ./examples/02-weather-update/server/main.go
