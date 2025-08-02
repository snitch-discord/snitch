package main

import (
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
	"golang.org/x/net/http2/h2c"
)

func main() {
	port := flag.Int("port", 4200, "port to listen on")
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

	// Create database service client (Connect RPC over HTTP)
	dbServiceURL := fmt.Sprintf("http://%s:%s", dbHost, dbPort)
	dbClient := snitchv1connect.NewDatabaseServiceClient(
		&http.Client{
			Timeout: 30 * time.Second,
		}, 
		dbServiceURL,
	)

	eventService := service.NewEventService(dbClient)
	registrar := service.NewRegisterServer(dbClient)
	reportServer := service.NewReportServer(dbClient, eventService)
	userServer := service.NewUserServer(dbClient)

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

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
		// No ReadTimeout/WriteTimeout for streaming support
	}

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
