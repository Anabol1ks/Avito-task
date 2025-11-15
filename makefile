run:
	go run ./cmd/app/main.go

doc:
	docker-compose up -d --build

doc-rebuild:
	docker-compose up -d --build app