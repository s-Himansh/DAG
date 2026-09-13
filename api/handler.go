package api

import (
	"context"
	"dag/store"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	store   store.WorkflowStore
	execCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewHandler(s store.WorkflowStore) *Handler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{
		store:   s,
		execCtx: ctx,
		cancel:  cancel,
	}
}

func (h *Handler) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	r.Get("/health", h.Health)
	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {})

	r.Route("/workflows", func(r chi.Router) {
		r.Post("/", h.SubmitWorkflow)
		r.Get("/", h.ListWorkflows)
		r.Get("/{id}", h.GetWorkflow)
		r.Get("/{id}/tasks", h.GetWorkflowTasks)
		r.Post("/{id}/cancel", h.CancelWorkflow)
	})

	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
	var req store.SubmitWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Tasks) == 0 {
		http.Error(w, `{"error":"at least one task is required"}`, http.StatusBadRequest)
		return
	}

	taskIDs := make(map[string]bool)
	for _, t := range req.Tasks {
		if t.ID == "" {
			http.Error(w, `{"error":"task id is required"}`, http.StatusBadRequest)
			return
		}
		if taskIDs[t.ID] {
			http.Error(w, `{"error":"duplicate task id"}`, http.StatusBadRequest)
			return
		}
		taskIDs[t.ID] = true
	}

	for _, t := range req.Tasks {
		for _, dep := range t.Dependencies {
			if !taskIDs[dep] {
				http.Error(w, `{"error":"dependency not found: `+dep+`"}`, http.StatusBadRequest)
				return
			}
		}
	}

	wf := h.store.CreateWorkflow(req)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.store.ExecuteWorkflow(h.execCtx, wf.ID)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wf)
}

func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	wf, ok := h.store.GetWorkflow(id)
	if !ok {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	wfList := h.store.ListWorkflows()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wfList)
}

func (h *Handler) GetWorkflowTasks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	wf, ok := h.store.GetWorkflow(id)
	if !ok {
		http.Error(w, `{"error":"workflow not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf.TaskStatuses)
}

func (h *Handler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !h.store.CancelWorkflow(id) {
		http.Error(w, `{"error":"workflow not found or cannot be cancelled"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func (h *Handler) Shutdown() {
	h.cancel()
	h.wg.Wait()
}
