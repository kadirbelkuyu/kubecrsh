.PHONY: local-test
local-test:
	@./scripts/quick-test.sh

.PHONY: check
check:
	@./scripts/pre-commit.sh

.PHONY: test-docker
test-docker:
	@./scripts/test-docker.sh

.PHONY: install-hooks
install-hooks:
	@cp scripts/pre-commit.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed"

.PHONY: deploy
deploy:
	kubectl apply -f manifests/

.PHONY: build
build:
	go build -o bin/kubecrsh ./cmd/kubecrsh

.PHONY: run-daemon
run-daemon: build
	./bin/kubecrsh daemon --http-addr :8080

.PHONY: run-watch
run-watch: build
	./bin/kubecrsh watch

.PHONY: docker-build
docker-build:
	docker build -t kubecrsh:latest .

.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: coverage
coverage: test
	go tool cover -html=coverage.out

.PHONY: lint
lint:
	golangci-lint run --timeout=5m

.PHONY: fmt
fmt:
	gofmt -w -s .
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf reports/
	rm -f coverage.out

.PHONY: undeploy
undeploy:
	kubectl delete -f manifests/

.DEFAULT_GOAL := build
