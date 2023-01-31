package sync

import (
	"net/http"

	"github.com/spf13/viper"
)

// StartServer starts an HTTP server that can be used
// to inspect the status of the current sync operations
func StartServer(pool *SyncerPool) {
	mux := http.DefaultServeMux
	h := Handler{
		Pool: pool,
	}

	mux.Handle("/list", &Handler{
		Pool:       pool,
		H:          h.List,
		Token:      viper.GetString("api.authToken"),
		EnableAuth: viper.GetBool("api.enableAuth"),
	})

	mux.Handle("/ops", &Handler{
		Pool:       pool,
		H:          h.Ops,
		Token:      viper.GetString("api.authToken"),
		EnableAuth: viper.GetBool("api.enableAuth"),
	})

	http.ListenAndServe(":"+viper.GetString("api.port"), mux)
}
