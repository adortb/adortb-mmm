package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adortb/adortb-mmm/internal/model"
	"github.com/adortb/adortb-mmm/internal/optimizer"
	"github.com/adortb/adortb-mmm/internal/reporting"
)

// ModelStatus 模型状态
type ModelStatus string

const (
	StatusTraining ModelStatus = "training"
	StatusReady    ModelStatus = "ready"
	StatusFailed   ModelStatus = "failed"
)

// ModelRecord 存储训练完成的模型
type ModelRecord struct {
	ID        int64
	Status    ModelStatus
	Model     *model.FittedModel
	History   []model.DataPoint
	CreatedAt time.Time
	Error     string
}

// Store 线程安全模型存储
type Store struct {
	mu      sync.RWMutex
	models  map[int64]*ModelRecord
	counter atomic.Int64
}

func NewStore() *Store {
	return &Store{models: make(map[int64]*ModelRecord)}
}

func (s *Store) Create(rec *ModelRecord) int64 {
	id := s.counter.Add(1)
	rec.ID = id
	s.mu.Lock()
	s.models[id] = rec
	s.mu.Unlock()
	return id
}

func (s *Store) Get(id int64) (*ModelRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.models[id]
	return rec, ok
}

func (s *Store) Update(id int64, fn func(*ModelRecord)) {
	s.mu.Lock()
	if rec, ok := s.models[id]; ok {
		fn(rec)
	}
	s.mu.Unlock()
}

// Handler HTTP 处理器
type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册路由到 mux
func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /v1/models/train", h.TrainModel)
	mux.HandleFunc("GET /v1/models/{id}", h.GetModel)
	mux.HandleFunc("POST /v1/predict", h.Predict)
	mux.HandleFunc("POST /v1/optimize", h.Optimize)
	mux.HandleFunc("GET /v1/contribution", h.Contribution)
}

// TrainRequest 训练请求
type TrainRequest struct {
	Channels     []string `json:"channels"`
	TargetMetric string   `json:"target_metric"`
	DateFrom     string   `json:"date_from"`
	DateTo       string   `json:"date_to"`
	Granularity  string   `json:"granularity"`
}

// TrainModel POST /v1/models/train
func (h *Handler) TrainModel(w http.ResponseWriter, r *http.Request) {
	var req TrainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误: "+err.Error())
		return
	}
	if len(req.Channels) == 0 {
		writeError(w, http.StatusBadRequest, "channels 不能为空")
		return
	}

	rec := &ModelRecord{
		Status:    StatusTraining,
		CreatedAt: time.Now(),
	}
	id := h.store.Create(rec)

	// 异步训练（使用 mock 数据）
	go func() {
		data := generateMockData(req.Channels, 104)
		cfg := model.DefaultFitConfig(req.Channels)
		fitted, err := model.Fit(data, cfg)
		h.store.Update(id, func(m *ModelRecord) {
			if err != nil {
				m.Status = StatusFailed
				m.Error = err.Error()
			} else {
				m.Status = StatusReady
				m.Model = fitted
				m.History = data
			}
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"model_id": id, "status": StatusTraining})
}

// GetModel GET /v1/models/:id
func (h *Handler) GetModel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 model_id")
		return
	}
	rec, ok := h.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "模型不存在")
		return
	}

	resp := map[string]any{
		"model_id":  rec.ID,
		"status":    rec.Status,
		"created_at": rec.CreatedAt,
	}
	if rec.Status == StatusReady && rec.Model != nil {
		params := make([]map[string]any, 0, len(rec.Model.BestParams))
		for _, p := range rec.Model.BestParams {
			params = append(params, map[string]any{
				"channel": p.Name,
				"lambda":  p.Lambda,
				"alpha":   p.Alpha,
				"gamma":   p.Gamma,
			})
		}
		resp["params"] = params
		resp["validation_mape"] = rec.Model.ValidationMAPE
		resp["r2"] = rec.Model.Regression.R2
	}
	if rec.Status == StatusFailed {
		resp["error"] = rec.Error
	}
	writeJSON(w, http.StatusOK, resp)
}

// PredictRequest 预测请求
type PredictHTTPRequest struct {
	ModelID      int64              `json:"model_id"`
	FutureSpends []map[string]float64 `json:"future_spends"` // 每周各渠道支出
}

// Predict POST /v1/predict
func (h *Handler) Predict(w http.ResponseWriter, r *http.Request) {
	var req PredictHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	rec, ok := h.store.Get(req.ModelID)
	if !ok || rec.Status != StatusReady {
		writeError(w, http.StatusNotFound, "模型不存在或未就绪")
		return
	}

	channels := make([]string, len(rec.Model.BestParams))
	for i, p := range rec.Model.BestParams {
		channels[i] = p.Name
	}

	futureSpends := make([][]float64, len(req.FutureSpends))
	for i, week := range req.FutureSpends {
		row := make([]float64, len(channels))
		for ci, ch := range channels {
			row[ci] = week[ch]
		}
		futureSpends[i] = row
	}

	preds, err := rec.Model.Predict(rec.History, futureSpends)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"predictions": preds})
}

// OptimizeHTTPRequest 优化请求
type OptimizeHTTPRequest struct {
	ModelID       int64              `json:"model_id"`
	TotalBudget   float64            `json:"total_budget"`
	HorizonWeeks  int                `json:"horizon_weeks"`
	PerChannelMin map[string]float64 `json:"per_channel_min"`
	PerChannelMax map[string]float64 `json:"per_channel_max"`
}

// Optimize POST /v1/optimize
func (h *Handler) Optimize(w http.ResponseWriter, r *http.Request) {
	var req OptimizeHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.TotalBudget <= 0 {
		writeError(w, http.StatusBadRequest, "total_budget 必须大于 0")
		return
	}
	if req.HorizonWeeks <= 0 {
		req.HorizonWeeks = 4
	}

	rec, ok := h.store.Get(req.ModelID)
	if !ok || rec.Status != StatusReady {
		writeError(w, http.StatusNotFound, "模型不存在或未就绪")
		return
	}

	result, err := optimizer.Optimize(optimizer.OptimizeRequest{
		FittedModel:  rec.Model,
		History:      rec.History,
		HorizonWeeks: req.HorizonWeeks,
		Constraints: optimizer.BudgetConstraints{
			TotalBudget:   req.TotalBudget,
			PerChannelMin: req.PerChannelMin,
			PerChannelMax: req.PerChannelMax,
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"allocation":           result.Allocation,
		"expected_conversions": result.ExpectedConversions,
		"lift_vs_equal":        result.LiftVsEqual,
	})
}

// Contribution GET /v1/contribution?model_id=&period=
func (h *Handler) Contribution(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("model_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 model_id")
		return
	}

	rec, ok := h.store.Get(id)
	if !ok || rec.Status != StatusReady {
		writeError(w, http.StatusNotFound, "模型不存在或未就绪")
		return
	}

	report, err := reporting.GenerateContribution(rec.Model, rec.History)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
