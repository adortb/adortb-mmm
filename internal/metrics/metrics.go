package metrics

import (
	"expvar"
	"net/http"
	"sync/atomic"
)

var (
	trainRequests    atomic.Int64
	predictRequests  atomic.Int64
	optimizeRequests atomic.Int64
)

func init() {
	expvar.Publish("mmm_train_requests", expvar.Func(func() any { return trainRequests.Load() }))
	expvar.Publish("mmm_predict_requests", expvar.Func(func() any { return predictRequests.Load() }))
	expvar.Publish("mmm_optimize_requests", expvar.Func(func() any { return optimizeRequests.Load() }))
}

func IncrTrain()    { trainRequests.Add(1) }
func IncrPredict()  { predictRequests.Add(1) }
func IncrOptimize() { optimizeRequests.Add(1) }

// MetricsHandler 暴露 /metrics 端点
func MetricsHandler() http.Handler {
	return expvar.Handler()
}

// HealthHandler 暴露 /health 端点
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}
