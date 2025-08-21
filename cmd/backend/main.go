package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"snitch/internal/backend/backendconfig"
	"snitch/internal/backend/service"
	"snitch/internal/backend/service/interceptor"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
)

func main() {
	port := flag.Int("port", 4200, "port to listen on")
	flag.Parse()

	config, err := backendconfig.FromEnv()
	if err != nil {
		log.Fatalf("Failed to load backend configuration from environment: %v", err)
	}

	// Load CA certificate for database service validation
	caCert, err := os.ReadFile(config.CaCertFilePath)
	if err != nil {
		log.Fatal("Failed to read CA certificate", "error", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatal("Failed to parse CA certificate")
	}

	// Create database service client (Connect RPC over HTTPS)
	dbServiceURL, err := config.DbURL()
	if err != nil {
		log.Fatalf("Failed to load db URL from environment: %v", err)
	}
	dbClient := snitchv1connect.NewDatabaseServiceClient(
		&http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		},
		dbServiceURL.String(),
	)

	eventService := service.NewEventService(dbClient, config.JwtSecret)
	registrar := service.NewRegisterServer(dbClient)
	reportServer := service.NewReportServer(dbClient, eventService)
	userServer := service.NewUserServer(dbClient)

	// Load TLS certificate for backend service
	cert, err := tls.LoadX509KeyPair(config.CertFilePath, config.KeyFilePath)
	if err != nil {
		log.Fatal("Failed to load TLS certificate", "error", err)
	}

	baseInterceptors := connect.WithInterceptors(
		interceptor.NewRecoveryInterceptor(),
		interceptor.NewLogInterceptor(),
		interceptor.NewAuthInterceptor(config.JwtSecret, dbClient),
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
		NextProtos:   []string{"h2"},
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		// No ReadTimeout/WriteTimeout for streaming support
	}

	slog.Info("Starting backend service with TLS", "port", *port, "db_url", dbServiceURL, "cert", config.CertFilePath)

	if err := server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
