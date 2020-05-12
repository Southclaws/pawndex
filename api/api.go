package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Masterminds/semver"
	"github.com/go-chi/chi"
	"github.com/gorilla/handlers"
	"go.uber.org/zap"

	"github.com/Southclaws/pawndex/storage"
)

func Run(store storage.Storer) error {
	router := chi.NewMux()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		all, err := store.GetAll()
		if err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(all); err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	router.Get("/package/{user}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		user := chi.URLParam(r, "user")
		repo := chi.URLParam(r, "repo")

		p, exists, err := store.Get(user, repo)
		if err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, "Package not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(p); err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	router.Get("/package/{user}/{repo}/latest", func(w http.ResponseWriter, r *http.Request) {
		user := chi.URLParam(r, "user")
		repo := chi.URLParam(r, "repo")

		p, exists, err := store.Get(user, repo)
		if err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !exists {
			http.Error(w, "Package not found", http.StatusNotFound)
			return
		}

		if len(p.Tags) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		latest, err := semver.NewVersion(p.Tags[0])
		if err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = w.Write([]byte{
			byte(latest.Major()),
			byte(latest.Minor()),
			byte(latest.Patch()),
		})
		if err != nil {
			zap.L().Error("failed to handle request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	s := http.Server{
		Addr: "0.0.0.0:8080",
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{
				"Cache-Control",
				"X-File-Name",
				"X-Requested-With",
				"X-File-Name",
				"Content-Type",
				"Authorization",
				"Set-Cookie",
				"Cookie",
			}),
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT"}),
			handlers.AllowCredentials(),
		)(router),
		IdleTimeout: time.Minute,
	}

	return s.ListenAndServe()
}
