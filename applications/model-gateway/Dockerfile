FROM docker.io/library/golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /gateway ./cmd/gateway
FROM docker.io/library/alpine:3.21
RUN addgroup -S gateway && adduser -S gateway -G gateway
COPY --from=build /gateway /gateway
USER gateway
EXPOSE 5000
ENTRYPOINT ["/gateway"]
