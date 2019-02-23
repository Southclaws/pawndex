package service

import (
	"encoding/json"
	"net/http"
	"net/http/pprof"

	"github.com/Masterminds/semver"
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
			zap.L().Error("failed to encode package list",
				zap.Error(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(contents)
		if err != nil {
			zap.L().Fatal("failed to write",
				zap.Error(err))
		}
	})
	router.HandleFunc("/package/{user}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		p, exists := app.getPackage(mux.Vars(r)["user"], mux.Vars(r)["repo"])
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(p)
		if err != nil {
			zap.L().Fatal("failed to write",
				zap.Error(err))
		}
	})
	router.HandleFunc("/package/{user}/{repo}/latest", func(w http.ResponseWriter, r *http.Request) {
		p, exists := app.getPackage(mux.Vars(r)["user"], mux.Vars(r)["repo"])
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if len(p.Tags) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		latest, err := semver.NewVersion(p.Tags[0])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte{
			byte(latest.Major()),
			byte(latest.Minor()),
			byte(latest.Patch()),
		})
	})
	router.Handle("/metrics", promhttp.Handler())

	router.HandleFunc("/debug/pprof/{v}", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	zap.L().Info("listening for http requests",
		zap.String("bind", app.config.Bind))

	err := http.ListenAndServe(app.config.Bind, handlers.CORS(
		handlers.AllowedHeaders([]string{"Cache-Control", "X-File-Name", "X-Requested-With", "X-File-Name", "Content-Type", "Authorization", "Set-Cookie", "Cookie"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT"}),
		handlers.AllowCredentials(),
	)(router))

	if err != nil {
		zap.L().Fatal("serve failed",
			zap.Error(err))
	}
}
