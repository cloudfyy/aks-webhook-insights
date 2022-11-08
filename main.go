package main

import (
	"aks-webhook-insights/akshook"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main(){
	var param akshook.AKSWebhookParameters
	flag.IntVar(&param.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&param.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&param.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(param.CertFile, param.KeyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
	}

	aksWebhookServer := &akshook.WebhookServer{
		 Server: &http.Server{
			Addr: fmt.Sprintf(":%v", param.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", aksWebhookServer.Serve)
	aksWebhookServer.Server.Handler = mux

	go func() {
		if err := aksWebhookServer.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	glog.Info("Server Started")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	err = aksWebhookServer.Server.Shutdown(context.Background())
	if err != nil {
		glog.Errorf("Shutdown server (Triggered by sys signal) Failed, Message: %s", err)
	}
}
