FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download || true
COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /tanchiki ./cmd/tanchiki

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=builder /tanchiki /tanchiki
EXPOSE 2222
ENTRYPOINT ["/tanchiki"]
