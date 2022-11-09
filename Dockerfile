FROM golang:1.18.4-alpine3.15 AS build
ADD . /aks-webhook-insights

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on

RUN cd /aks-webhook-insights && go build -o aksWebhook

FROM alpine:latest as webhook
WORKDIR /app
COPY --from=build /aks-webhook-insights/aksWebhook /app
ENTRYPOINT ./aksWebhook