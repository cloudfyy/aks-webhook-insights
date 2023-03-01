package main

import (
	"aks-webhook-insights/akshook"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

var ErrEnvVarEmpty = errors.New("getenv: environment variable empty")

func getenvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, ErrEnvVarEmpty
	}
	return v, nil
}

func getenvInt(key string) (int, error) {
	s, err := getenvStr(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func getenvBool(key string) (bool, error) {
	s, err := getenvStr(key)
	if err != nil {
		return false, err
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, err
	}
	return v, nil
}

func main() {
	// init klog
	klog.InitFlags(nil)

	port, err := getenvInt("SERVERPORT")
	if err != nil {
		klog.Fatal(err)
	}
	var param akshook.AksWebhookParam
	flag.IntVar(&param.Port, "port", port, "Webhook server port.")
	flag.StringVar(&param.CertFile, "tlsCertFile", "/mnt/webhook/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&param.KeyFile, "tlsKeyFile", "/mnt/webhook/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(param.CertFile, param.KeyFile)
	if err != nil {
		klog.Errorf("Failed to load key pair: %v", err)
	}

	klog.Info("port: ", param.Port, ", certFile: ", param.CertFile, ", keyFile: ", param.KeyFile)
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

		klog.Info("Server Started")
		if err := aksWebhookServer.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	catchSystemStopSignal(aksWebhookServer)
	select {}
}
