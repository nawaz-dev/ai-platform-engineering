FROM alpine:3.20

RUN apk add --no-cache git curl jq ca-certificates

USER 1001