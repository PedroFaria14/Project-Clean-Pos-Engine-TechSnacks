.PHONY: up down build test lint

up:
	docker-compose up -d

down:
	docker-compose down

build:
	docker-compose build

test-backend:
	cd backend && go test ./... -v

fmt:
	cd backend && go fmt ./...


commit:
	git add .
	git commit -m "$(m)"