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
	CGO_ENABLED=1 go build -trimpath -o ./bin/01-hello-world-server ./examples/01-hello-world/server/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/01-hello-world-client ./examples/01-hello-world/client/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/02-weather-update-server ./examples/02-weather-update/server/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/02-weather-update-client ./examples/02-weather-update/client/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/03-parallel-task-ventilator ./examples/03-parallel-task/ventilator/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/03-parallel-task-worker ./examples/03-parallel-task/worker/main.go
	CGO_ENABLED=1 go build -trimpath -o ./bin/03-parallel-task-sync ./examples/03-parallel-task/sync/main.go
