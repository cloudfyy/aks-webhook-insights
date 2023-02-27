package main

import (
	"aks-webhook-insights/akshook"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
)

func catchSystemStopSignal(server *akshook.WebhookServer) {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill, syscall.SIGQUIT)
	go func() {
		<-s
		server.Server.Shutdown(context.Background())
	}()
}

func main() {
	// init klog
	klog.InitFlags(nil)

	var param akshook.AksWebhookParam
	flag.IntVar(&param.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&param.CertFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&param.KeyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(param.CertFile, param.KeyFile)
	if err != nil {
		klog.Errorf("Failed to load key pair: %v", err)
	}

	klog.Info("port: %v, certFile: %v, keyFile: %v",
		param.Port, param.CertFile, param.KeyFile)
	aksWebhookServer := &akshook.WebhookServer{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", param.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", aksWebhookServer.Handler)
	aksWebhookServer.Server.Handler = mux

	go func() {
		klog.Info("port: %v, certFile: %v, keyFile: %v",
			param.Port, param.CertFile, param.KeyFile)
		klog.Info("Server Started")
		if err := aksWebhookServer.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	catchSystemStopSignal(aksWebhookServer)
	select {}
}
