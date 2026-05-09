FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/platform-api ./cmd/platform-api

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates && \
    addgroup -S forge && adduser -S -G forge forge && \
    mkdir -p /var/lib/forge/repos && chown -R forge:forge /var/lib/forge
COPY --from=build /out/platform-api /usr/local/bin/platform-api
USER forge
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/platform-api"]
