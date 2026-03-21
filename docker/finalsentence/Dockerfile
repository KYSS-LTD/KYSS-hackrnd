FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/finalsentence ./cmd/finalsentence

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/finalsentence /usr/local/bin/finalsentence
USER app
EXPOSE 2222
ENTRYPOINT ["/usr/local/bin/finalsentence"]
