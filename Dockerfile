FROM golang:1.18.4-alpine3.15 AS build
ADD . /aks-webhook-insights
RUN cd /aks-webhook-insights && go build -o aksWebhook

FROM alpine:latest
WORKDIR /app
COPY --from=build /aks-webhook-insights /app
ENTRYPOINT ./aksWebhook