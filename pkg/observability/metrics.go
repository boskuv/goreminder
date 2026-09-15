package observability

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(MetricsAddr string) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(MetricsAddr, nil); err != nil && err != http.ErrServerClosed {
			// Metrics are best-effort; surface the failure on stderr for ops visibility.
			_, _ = os.Stderr.WriteString("metrics server stopped: " + err.Error() + "\n")
		}
	}()
}
