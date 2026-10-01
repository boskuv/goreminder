package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CalendarSyncSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "goreminder_calendar_sync_success_total",
		Help: "Successful calendar binding sync operations",
	})
	CalendarSyncErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "goreminder_calendar_sync_errors_total",
		Help: "Failed calendar binding sync operations",
	})
	CalendarOutboxDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "goreminder_calendar_outbox_depth",
		Help: "Pending/processing sync outbox items",
	})
	CalendarOutboxProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goreminder_calendar_outbox_processed_total",
		Help: "Processed sync outbox items by result",
	}, []string{"result"})
)
