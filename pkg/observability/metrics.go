package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9191", nil)
	}()
}
