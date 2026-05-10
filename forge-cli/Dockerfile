FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/codecollab-co/forge/forge-cli/internal/cli.Version=${VERSION}" \
    -o /out/forge ./cmd/forge

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/forge /usr/local/bin/forge
ENTRYPOINT ["/usr/local/bin/forge"]
