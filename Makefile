.PHONY: build build-hub build-snake run-hub run-snake docker-build docker-up docker-down test

build: build-hub build-snake

build-hub:
	go build -o bin/hub ./cmd/hub

build-snake:
	go build -o bin/snake ./cmd/snake

run-hub:
	LISTEN_ADDR=:2222 SNAKE_IMAGE=ssh-games-snake:latest DOCKER_NETWORK=games-net go run ./cmd/hub

run-snake:
	LISTEN_ADDR=:2223 go run ./cmd/snake

docker-build:
	docker build -f docker/snake/Dockerfile -t ssh-games-snake:latest .
	docker build -f docker/hub/Dockerfile -t ssh-games-hub:latest .

docker-up: docker-build
	docker network create games-net 2>/dev/null || true
	cd manifests && docker compose up -d

docker-down:
	cd manifests && docker compose down
	docker ps -q --filter "label=ssh-games=true" | xargs -r docker rm -f

test:
	go test ./...
