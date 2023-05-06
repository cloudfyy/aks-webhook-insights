FROM golang:1.20.2-alpine3.17 AS build
ADD . /aks-webhook-insights

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on

RUN cd /aks-webhook-insights && go install golang.org/x/exp/slices && go build -o aksWebhook

FROM alpine:latest as webhook
ENV SERVERPORT=1337
WORKDIR /app
COPY --from=build /aks-webhook-insights/aksWebhook /app
ENTRYPOINT ./aksWebhook