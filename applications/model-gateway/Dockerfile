FROM docker.io/library/golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /gateway ./cmd/gateway
FROM docker.io/library/alpine:3.21
RUN addgroup -S gateway && adduser -S gateway -G gateway
WORKDIR /app
COPY --from=build /gateway /app/gateway
COPY configs/models.json /app/configs/models.json
USER gateway
EXPOSE 5000
ENTRYPOINT ["/app/gateway"]
