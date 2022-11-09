FROM golang:1.18.4-alpine3.15 AS build
WORKDIR /Workspace
COPY main.go main.go
COPY go.mod go.mod
COPY go.sum go.sum
COPY util/ util/

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on

RUN go build -o aksWebhookCert

FROM alpine:latest as tlsgenerator
WORKDIR /app
COPY --from=build /Workspace/aksWebhookCert /app
ENTRYPOINT ./aksWebhookCert