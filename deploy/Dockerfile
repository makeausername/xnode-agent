# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/xnode ./cmd/xnode

FROM alpine:3.20

COPY --from=build /out/xnode /usr/local/bin/xnode

VOLUME ["/var/lib/xnode", "/var/log/xnode"]

ENTRYPOINT ["/usr/local/bin/xnode"]
