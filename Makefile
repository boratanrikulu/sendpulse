.PHONY: all
all: build

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build: generate
	go build -o build/sendpulse ./cmd/sendpulse

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	go vet ./...
	go fmt ./...

.PHONY: docker
docker: build
	docker build -t sendpulse -f containers/images/sendpulse.dockerfile .

.PHONY: run-dev-db
run-dev-db:
	docker-compose -f containers/composes/dc.dev.yml up --build postgres

.PHONY: run-dev-srv
run-dev-srv: docker
	docker-compose -f containers/composes/dc.dev.yml up --build sendpulse

.PHONY: run-dev
run-dev: docker
	docker-compose -f containers/composes/dc.dev.yml up --build

.PHONY: clean-dev
clean-dev:
	docker-compose -f containers/composes/dc.dev.yml down --volumes --remove-orphans

.PHONY: clean
clean:
	rm -rf build/
