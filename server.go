package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func (app *App) runServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contents, err := json.Marshal(app.getPackageList())
		if err != nil {
			logger.Error("failed to encode package list",
				zap.Error(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(contents)
		if err != nil {
			logger.Fatal("failed to write",
				zap.Error(err))
		}
	})
	router.Handle("/metrics", promhttp.Handler())

	logger.Info("listening for http requests",
		zap.String("bind", app.config.Bind))

	err := http.ListenAndServe(app.config.Bind, handlers.CORS(
		handlers.AllowedHeaders([]string{"Cache-Control", "X-File-Name", "X-Requested-With", "X-File-Name", "Content-Type", "Authorization", "Set-Cookie", "Cookie"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT"}),
		handlers.AllowCredentials(),
	)(router))

	if err != nil {
		logger.Fatal("serve failed",
			zap.Error(err))
	}
}
