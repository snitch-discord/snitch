package main

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"snitch/internal/backend/dbconfig"
	"snitch/internal/backend/jwt"
	"snitch/internal/backend/metadata"
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

	libSQLConfig, err := dbconfig.LibSQLConfigFromEnv()
	if err != nil {
		panic(err)
	}

	pemKey, err := base64.StdEncoding.DecodeString(libSQLConfig.AuthKey)
	if err != nil {
		panic(err)
	}
	block, _ := pem.Decode([]byte(pemKey))
	parseResult, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	key := parseResult.(ed25519.PrivateKey)

	jwtDuration := 10 * time.Minute
	jwtCache := &jwt.TokenCache{}
	jwt.StartGenerator(jwtDuration, jwtCache, key)

	dbJwt, err := jwt.CreateToken(key)
	if err != nil {
		panic(err)
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	metadataDb, err := metadata.NewMetadataDB(dbCtx, dbJwt, libSQLConfig)
	if err != nil {
		panic(err)
	}
	defer metadataDb.Close()

	if err := metadataDb.PingContext(dbCtx); err != nil {
		panic(err)
	}

	eventService := service.NewEventService()
	registrar := service.NewRegisterServer(jwtCache, metadataDb, libSQLConfig)
	reportServer := service.NewReportServer(jwtCache, libSQLConfig, eventService)
	userServer := service.NewUserServer(jwtCache, libSQLConfig)

	baseInterceptors := connect.WithInterceptors(
		interceptor.NewRecoveryInterceptor(),
		interceptor.NewLogInterceptor(),
		interceptor.NewTraceInterceptor(),
	)

	mux := http.NewServeMux()
	mux.Handle(snitchv1connect.NewRegistrarServiceHandler(registrar, baseInterceptors))
	mux.Handle(snitchv1connect.NewReportServiceHandler(reportServer, baseInterceptors, connect.WithInterceptors(interceptor.NewGroupContextInterceptor(metadataDb))))
	mux.Handle(snitchv1connect.NewUserHistoryServiceHandler(userServer, baseInterceptors, connect.WithInterceptors(interceptor.NewGroupContextInterceptor(metadataDb))))
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
