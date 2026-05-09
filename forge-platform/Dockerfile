FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/platform-api ./cmd/platform-api

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/platform-api /platform-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/platform-api"]
