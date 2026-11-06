package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}

type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

// NewInMemoryTaskRepo создаёт потокобезопасный репозиторий
func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}

// Create создаёт новую задачу
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)

	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	id := strconv.FormatUint(r.seq, 10)

	task := Task{
		ID:        id,
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[id] = task

	return task, nil
}

// Get возвращает задачу по ID
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, false
	}

	return task, true
}

// List возвращает копию списка задач с заданным статусом
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Task, 0)

	for _, task := range r.tasks {
		if task.Done == done {
			result = append(result, task)
		}
	}

	return result
}

// SetDone изменяет статус выполнения задачи
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done != done {
		task.Done = done
		task.UpdatedAt = r.clock.Now()
		r.tasks[id] = task
	}

	return task, nil
}

// HTTP-обработчик
type httpHandler struct {
	repo TaskRepo
}

// NewHTTPHandler создаёт HTTP-обработчик
func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{
		repo: repo,
	}
}

// ServeHTTP маршрутизирует HTTP-запросы
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/tasks" {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if !strings.HasPrefix(path, "/tasks/") {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(path, "/tasks/")

	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, id)

	case http.MethodPatch:
		h.handlePatch(w, r, id)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreate обрабатывает POST /tasks
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if request.Title == nil {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*request.Title)
	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// handleGet обрабатывает GET /tasks/{id}
func (h *httpHandler) handleGet(w http.ResponseWriter, id string) {
	task, ok := h.repo.Get(id)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleList обрабатывает GET /tasks
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false

	values, exists := r.URL.Query()["done"]

	if exists {
		// Параметр должен присутствовать ровно один раз.
		if len(values) != 1 {
			http.Error(w, "invalid done parameter", http.StatusBadRequest)
			return
		}

		switch values[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			http.Error(w, "invalid done parameter", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}

		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	if tasks == nil {
		tasks = make([]Task, 0)
	}

	writeJSON(w, http.StatusOK, tasks)
}

// handlePatch обрабатывает PATCH /tasks/{id}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if request.Done == nil {
		http.Error(w, "done is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *request.Done)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// decodeStrictJSON декодирует ровно один JSON-объект
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	var extra json.RawMessage

	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}

		return err
	}

	return nil
}

// writeJSON записывает JSON-ответ с заданным HTTP-статусом
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}
