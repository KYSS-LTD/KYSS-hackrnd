.PHONY: build build-hub build-snake build-pingpong build-finalsentence run-hub run-snake run-pingpong run-finalsentence docker-build docker-up docker-down test

build: build-hub build-snake build-pingpong build-finalsentence

build-hub:
	go build -o bin/hub ./cmd/hub

build-snake:
	go build -o bin/snake ./cmd/snake

build-pingpong:
	go build -o bin/pingpong ./cmd/pingpong

build-finalsentence:
	go build -o bin/finalsentence ./cmd/finalsentence

run-hub:
	LISTEN_ADDR=:2222 SNAKE_IMAGE=ssh-games-snake:latest PINGPONG_IMAGE=ssh-games-pingpong:latest FINALSENTENCE_IMAGE=ssh-games-finalsentence:latest DOCKER_NETWORK=games-net go run ./cmd/hub

run-snake:
	LISTEN_ADDR=:2223 go run ./cmd/snake

run-pingpong:
	LISTEN_ADDR=:2224 go run ./cmd/pingpong

run-finalsentence:
	LISTEN_ADDR=:2225 go run ./cmd/finalsentence

docker-build:
	docker build -f docker/snake/Dockerfile -t ssh-games-snake:latest .
	docker build -f docker/pingpong/Dockerfile -t ssh-games-pingpong:latest .
	docker build -f docker/finalsentence/Dockerfile -t ssh-games-finalsentence:latest .
	docker build -f docker/hub/Dockerfile -t ssh-games-hub:latest .

docker-up: docker-build
	docker network create games-net 2>/dev/null || true
	cd manifests && docker compose up -d

docker-down:
	cd manifests && docker compose down
	docker ps -q --filter "label=ssh-games=true" | xargs -r docker rm -f

test:
	go test ./...
