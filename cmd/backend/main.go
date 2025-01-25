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
	"snitch/internal/backend/handler"
	"snitch/internal/backend/jwt"
	"snitch/internal/backend/metadata"
	"snitch/internal/backend/middleware"
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

	reportEndpointHandler := handler.CreateReportHandler(jwtCache, libSQLConfig)
	databaseEndpointHandler := handler.CreateRegistrationHandler(jwtCache, metadataDb, libSQLConfig)

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/databases":
			databaseEndpointHandler(w, r)
		case "/reports":
			middleware.GroupContext(reportEndpointHandler, metadataDb)(w, r)
		default:
			http.Error(w, "404 Not Found", http.StatusNotFound)
		}
	}

	handler = middleware.RecordResponse(handler)
	handler = middleware.Recovery(handler)
	handler = middleware.PermissiveCORSHandler(handler)
	handler = middleware.Log(handler)
	handler = middleware.Trace(handler)

	server := http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           handler,
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		ReadHeaderTimeout: 200 * time.Millisecond,
	}

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}
