package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"snitch/internal/backend/service"
	"snitch/internal/backend/service/interceptor"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
)

func main() {
	port := flag.Int("port", 4200, "port to listen on")
	certFile := flag.String("cert", "./certs/backend/cert.pem", "TLS certificate file")
	keyFile := flag.String("key", "./certs/backend/key.pem", "TLS private key file")
	caCertFile := flag.String("ca-cert", "./certs/ca/ca-cert.pem", "CA certificate file for validating database service")
	flag.Parse()

	// Get database service connection details from environment
	dbHost := os.Getenv("SNITCH_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("SNITCH_DB_PORT")
	if dbPort == "" {
		dbPort = "5200"
	}

	// Load CA certificate for database service validation
	caCert, err := os.ReadFile(*caCertFile)
	if err != nil {
		slog.Error("Failed to read CA certificate", "error", err)
		os.Exit(1)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		slog.Error("Failed to parse CA certificate")
		os.Exit(1)
	}

	// Create database service client (Connect RPC over HTTPS)
	dbServiceURL := fmt.Sprintf("https://%s:%s", dbHost, dbPort)
	dbClient := snitchv1connect.NewDatabaseServiceClient(
		&http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}, 
		dbServiceURL,
	)

	eventService := service.NewEventService(dbClient)
	registrar := service.NewRegisterServer(dbClient)
	reportServer := service.NewReportServer(dbClient, eventService)
	userServer := service.NewUserServer(dbClient)

	// Load TLS certificate for backend service
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		slog.Error("Failed to load TLS certificate", "error", err)
		os.Exit(1)
	}

	baseInterceptors := connect.WithInterceptors(
		interceptor.NewRecoveryInterceptor(),
		interceptor.NewLogInterceptor(),
		interceptor.NewTraceInterceptor(),
	)

	mux := http.NewServeMux()
	mux.Handle(snitchv1connect.NewRegistrarServiceHandler(registrar, baseInterceptors))
	mux.Handle(snitchv1connect.NewReportServiceHandler(reportServer, baseInterceptors))
	mux.Handle(snitchv1connect.NewUserHistoryServiceHandler(userServer, baseInterceptors))
	mux.Handle(snitchv1connect.NewEventServiceHandler(eventService, baseInterceptors))

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		// No ReadTimeout/WriteTimeout for streaming support
	}

	// Configure HTTP/2 explicitly
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		slog.Error("Failed to configure HTTP/2", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting backend service with TLS", "port", *port, "db_url", dbServiceURL, "cert", *certFile)

	if err := server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
