package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer() (*httptest.Server, *Store) {
	store := NewStore()
	handler := NewHandler(store)
	mux := http.NewServeMux()
	RegisterRoutes(mux, handler)
	return httptest.NewServer(mux), store
}

func TestTrainModel_Success(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	body, _ := json.Marshal(TrainRequest{
		Channels:     []string{"search", "display"},
		TargetMetric: "revenue",
		DateFrom:     "2025-01-01",
		DateTo:       "2026-01-01",
	})
	resp, err := http.Post(srv.URL+"/v1/models/train", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if _, ok := result["model_id"]; !ok {
		t.Error("response missing model_id")
	}
}

func TestTrainModel_EmptyChannels(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	body, _ := json.Marshal(TrainRequest{Channels: nil})
	resp, _ := http.Post(srv.URL+"/v1/models/train", "application/json", bytes.NewReader(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetModel_NotFound(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/v1/models/999")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestGetModel_Training(t *testing.T) {
	srv, store := newTestServer()
	defer srv.Close()

	rec := &ModelRecord{Status: StatusTraining, CreatedAt: time.Now()}
	id := store.Create(rec)

	resp, _ := http.Get(srv.URL + "/v1/models/1")
	defer resp.Body.Close()

	_ = id
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestOptimize_ModelNotReady(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	body, _ := json.Marshal(OptimizeHTTPRequest{
		ModelID:     999,
		TotalBudget: 100000,
	})
	resp, _ := http.Post(srv.URL+"/v1/optimize", "application/json", bytes.NewReader(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestContribution_InvalidModelID(t *testing.T) {
	srv, _ := newTestServer()
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/v1/contribution?model_id=abc")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
