FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download || true
COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /finalsentence ./cmd/finalsentence

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=builder /finalsentence /finalsentence
EXPOSE 2222
ENTRYPOINT ["/finalsentence"]
