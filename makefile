run:
	go run ./cmd/app/main.go

doc:
	docker-compose up -d --build

doc-rebuild:
	docker-compose up -d --build app

.PHONY: test
test:
	go test -v ./test/... -count=1

k6-test:
	k6 run k6/load_test.js

lint:
	golangci-lint run ./... -v