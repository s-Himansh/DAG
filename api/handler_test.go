package api

import (
	"bytes"
	"dag/store"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setupTestServer() *httptest.Server {
	s := store.NewMemoryStore()
	h := NewHandler(s)
	r := h.Router()
	return httptest.NewServer(r)
}

func TestHealthEndpoint(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
}

func TestSubmitWorkflow(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
			{ID: "task2", Execute: "fn2", Dependencies: []string{"task1"}},
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var wf store.Workflow
	json.NewDecoder(resp.Body).Decode(&wf)

	if wf.ID == "" {
		t.Error("expected workflow ID")
	}
	if wf.Name != "test-workflow" {
		t.Errorf("expected name test-workflow, got %v", wf.Name)
	}
	if wf.Status != store.StatusPending && wf.Status != store.StatusRunning {
		t.Errorf("expected status pending or running, got %v", wf.Status)
	}
}

func TestSubmitWorkflowInvalidBody(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader([]byte("invalid")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestSubmitWorkflowNoName(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestSubmitWorkflowNoTasks(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name:  "test-workflow",
		Tasks: []store.TaskDefinition{},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetWorkflow(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, _ := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	var wf store.Workflow
	json.NewDecoder(resp.Body).Decode(&wf)
	resp.Body.Close()

	resp, err := http.Get(ts.URL + "/workflows/" + wf.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var fetched store.Workflow
	json.NewDecoder(resp.Body).Decode(&fetched)

	if fetched.ID != wf.ID {
		t.Errorf("expected ID %s, got %s", wf.ID, fetched.ID)
	}
}

func TestGetWorkflowNotFound(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/workflows/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestListWorkflows(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
		},
	}
	body, _ := json.Marshal(reqBody)
	http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))

	resp, err := http.Get(ts.URL + "/workflows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var wfList []store.Workflow
	json.NewDecoder(resp.Body).Decode(&wfList)

	if len(wfList) < 1 {
		t.Errorf("expected at least 1 workflow, got %d", len(wfList))
	}
}

func TestGetWorkflowTasks(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
			{ID: "task2", Execute: "fn2"},
		},
	}
	body, _ := json.Marshal(reqBody)
	resp, _ := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	var wf store.Workflow
	json.NewDecoder(resp.Body).Decode(&wf)
	resp.Body.Close()

	resp, err := http.Get(ts.URL + "/workflows/" + wf.ID + "/tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var tasks map[string]*store.TaskStatus
	json.NewDecoder(resp.Body).Decode(&tasks)

	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestCancelWorkflow(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
		},
	}
	body, _ := json.Marshal(reqBody)
	resp, _ := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	var wf store.Workflow
	json.NewDecoder(resp.Body).Decode(&wf)
	resp.Body.Close()

	resp, err := http.Post(ts.URL+"/workflows/"+wf.ID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestWorkflowExecution(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	reqBody := store.SubmitWorkflowRequest{
		Name: "test-workflow",
		Tasks: []store.TaskDefinition{
			{ID: "task1", Execute: "fn1"},
			{ID: "task2", Execute: "fn2", Dependencies: []string{"task1"}},
		},
	}
	body, _ := json.Marshal(reqBody)
	resp, _ := http.Post(ts.URL+"/workflows", "application/json", bytes.NewReader(body))
	var wf store.Workflow
	json.NewDecoder(resp.Body).Decode(&wf)
	resp.Body.Close()

	time.Sleep(500 * time.Millisecond)

	resp, err := http.Get(ts.URL + "/workflows/" + wf.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	var fetched store.Workflow
	json.NewDecoder(resp.Body).Decode(&fetched)

	if fetched.Status != store.StatusCompleted {
		t.Errorf("expected status completed, got %v", fetched.Status)
	}
}
