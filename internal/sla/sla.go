// Package sla computes RPO/RTO metrics from the control DB and Prometheus
// metrics, producing a human-readable summary.
package sla

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Report summarizes sync lag and projected RTO.
type Report struct {
	PGLagP50       time.Duration
	PGLagP99       time.Duration
	LastSweepAge   time.Duration
	DriftCount     int64
	ComponentsHealthy int
	ComponentsTotal   int
}

// Generate collects current metric values and writes a summary to w.
func Generate(ctx context.Context, w io.Writer) error {
	report := &Report{
		ComponentsTotal: 4,
	}

	// Gather current metric values from the default registry.
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return fmt.Errorf("gather metrics: %w", err)
	}

	for _, mf := range mfs {
		switch mf.GetName() {
		case "geo_sync_pg_replay_lag_seconds":
			for _, m := range mf.GetMetric() {
				if m.GetGauge() != nil {
					v := m.GetGauge().GetValue()
					if v > float64(report.PGLagP50.Seconds()) {
						report.PGLagP50 = time.Duration(v * float64(time.Second))
					}
					if v > float64(report.PGLagP99.Seconds()) {
						report.PGLagP99 = report.PGLagP50
					}
					if v >= 0 {
						report.ComponentsHealthy++
					}
				}
			}
		case "geo_sync_drift_total":
			for _, m := range mf.GetMetric() {
				if m.GetCounter() != nil {
					report.DriftCount += int64(m.GetCounter().GetValue())
				}
			}
		case "geo_sync_last_sync_timestamp_seconds":
			for _, m := range mf.GetMetric() {
				if m.GetGauge() != nil {
					ts := m.GetGauge().GetValue()
					if ts > 0 {
						age := time.Since(time.Unix(int64(ts), 0))
						if age > report.LastSweepAge {
							report.LastSweepAge = age
						}
					}
				}
			}
		}
	}

	report.Print(w)
	return nil
}

// Print writes the SLA report to w.
func (r *Report) Print(w io.Writer) {
	fmt.Fprintf(w, "=== gitlab-geo-sync SLA Report ===\n\n")
	fmt.Fprintf(w, "PostgreSQL Replay Lag:\n")
	fmt.Fprintf(w, "  Current: %s\n", r.PGLagP50)
	fmt.Fprintf(w, "  Peak:    %s\n", r.PGLagP99)
	fmt.Fprintf(w, "\nLast Sync Age (oldest component): %s\n", r.LastSweepAge)
	fmt.Fprintf(w, "\nDrift Events (cumulative): %d\n", r.DriftCount)
	fmt.Fprintf(w, "Components Healthy: %d/%d\n", r.ComponentsHealthy, r.ComponentsTotal)
	fmt.Fprintf(w, "\nRPO Estimate: %s (PG replay lag)\n", r.PGLagP50)
	fmt.Fprintf(w, "RTO Estimate: ~2-5 min (pg_ctl promote + gitlab-ctl restart)\n")
	fmt.Fprintf(w, "\nNote: RPO for in-flight Sidekiq jobs is ~last dequeue time.\n")
}

// _ keeps the dto import for future metric parsing extensions.
var _ = dto.MetricFamily{}