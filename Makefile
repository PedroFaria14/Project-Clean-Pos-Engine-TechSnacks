.PHONY: up down build test test-backend test-frontend lint migrate e2e

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose build

migrate:
	docker compose run --rm --no-deps backend /usr/local/bin/migrate

test: test-backend test-frontend

test-frontend:
	cd frontend && npm test

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

e2e:
	cd frontend && npm run test:e2e

test-backend:
	cd backend && go test ./... -v

fmt:
	cd backend && go fmt ./...


commit:
	git add .
	git commit -m "$(m)"
