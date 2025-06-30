package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(MetricsAddr string) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(MetricsAddr, nil)
	}()
}
