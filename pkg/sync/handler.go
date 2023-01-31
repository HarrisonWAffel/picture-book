package sync

import (
	"fmt"
	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

type Handler struct {
	H          func(http.ResponseWriter, *http.Request)
	Pool       *SyncerPool
	Token      string
	EnableAuth bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.EnableAuth {
		authHeader := w.Header().Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token := strings.ReplaceAll(authHeader, "Bearer: ", "")
		if token != h.Token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	h.H(w, r)
}

// list returns the list of sync'ers currently running.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	switch q.Get("type") {
	case "configured":
		conf, err := ListConfiguredRegistrySyncers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(conf))

	case "active":
		active, err := ListActiveRegistrySyncers(h.Pool.Syncers)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(active))
	}
}

// Ops is an endpoint which lets you configure a currently running sync operation.
// mostly good for getting info on the currently running config, and stopping/starting
// already defined sync'ers. Maybe add new ones via this endpoint idk yet. Changes made via
// this endpoint don't persist across application executions.
func (h *Handler) Ops(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	syncName := q.Get("sync")
	h.Pool.Lock()
	defer h.Pool.Unlock()
	s, syncerFound := h.Pool.Syncers[syncName]

	switch q.Get("action") {
	case "changePeriod":
		if !syncerFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// todo;
		s.ChangePeriod("")
		w.Write([]byte("OK!"))

	case "details":
		if !syncerFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(s.Info())

	case "pause":
		if !syncerFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		err := PauseRegistry(s, h.Pool.CronJobScheduler)
		if err != nil {
			pkg.Logger.Errorf("error encountered pausing registry: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		delete(h.Pool.Syncers, syncName)
		pkg.Logger.Infof("Syncer for %s has been paused", syncName)
		w.Write([]byte(fmt.Sprintf("OK. %s has been paused.", syncName)))

	case "resume":
		syncer, job, err := ResumeRegistry(syncName, h.Pool)
		if err != nil {
			if errors.Is(err, pkg.RegistryNotFound) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.Write([]byte(err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		h.Pool.Syncers[syncName] = syncer
		pkg.Logger.Infof("Syncer for %s has been resumed", syncName)
		w.Write([]byte(fmt.Sprintf("OK. Syncer for %s is now running. Next execution will be at %s", syncName, job.NextRun().Format(pkg.TimeFormat))))
	}
}
