package dataserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ─── Types ───────────────────────────────────────────────────────────────────

// Field describes a single column in a resource.
type Field struct {
	Name string `json:"name"`
}

// ResourceSummary is a lightweight view of a resource (no row data).
type ResourceSummary struct {
	ID        uuid.UUID      `json:"id"`
	Label     string         `json:"label"`
	MimeType  string         `json:"mimeType"`
	Columns   []Field        `json:"columns"`
	RowCount  int            `json:"rowCount"`
	Pinned    bool           `json:"pinned"`
	MaxAge    *time.Duration `json:"maxAge,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// Resource holds full metadata plus all row data.
type Resource struct {
	ID        uuid.UUID
	Label     string
	MimeType  string
	Columns   []Field
	Rows      [][]string
	RowCount  int
	Pinned    bool
	MaxAge    *time.Duration
	CreatedAt time.Time
}

// Store is an in-memory, TTL-aware resource store.
type Store struct {
	mu      sync.RWMutex
	items   map[uuid.UUID]*item
	defTTL  time.Duration
	gcTick  time.Duration
	stopGC  chan struct{}
}

type item struct {
	res    *Resource
	expiry time.Time // zero = never expires (pinned or no TTL)
}

// NewStore creates a Store and starts its background GC loop.
func NewStore(defaultTTL, gcInterval time.Duration) *Store {
	s := &Store{
		items:  make(map[uuid.UUID]*item),
		defTTL: defaultTTL,
		gcTick: gcInterval,
		stopGC: make(chan struct{}),
	}
	go s.gcLoop()
	return s
}

// Stop terminates the background GC goroutine.
func (s *Store) Stop() { close(s.stopGC) }

func (s *Store) gcLoop() {
	ticker := time.NewTicker(s.gcTick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.reap()
		case <-s.stopGC:
			return
		}
	}
}

func (s *Store) reap() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, it := range s.items {
		if !it.expiry.IsZero() && now.After(it.expiry) {
			delete(s.items, id)
		}
	}
}

// Put inserts a new resource and returns it.
func (s *Store) Put(label, mimeType string, columns []Field, rows [][]string) (*Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := &Resource{
		ID:        uuid.New(),
		Label:     label,
		MimeType:  mimeType,
		Columns:   columns,
		Rows:      rows,
		RowCount:  len(rows),
		CreatedAt: time.Now(),
	}
	s.items[res.ID] = &item{
		res:    res,
		expiry: time.Now().Add(s.defTTL),
	}
	return res, nil
}

// List returns summaries of all non-expired resources.
func (s *Store) List() []ResourceSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ResourceSummary, 0, len(s.items))
	for _, it := range s.items {
		r := it.res
		out = append(out, ResourceSummary{
			ID:        r.ID,
			Label:     r.Label,
			MimeType:  r.MimeType,
			Columns:   r.Columns,
			RowCount:  r.RowCount,
			Pinned:    r.Pinned,
			MaxAge:    r.MaxAge,
			CreatedAt: r.CreatedAt,
		})
	}
	return out
}

// Get retrieves a resource by ID.
func (s *Store) Get(id uuid.UUID) (*Resource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	it, ok := s.items[id]
	if !ok {
		return nil, false
	}
	// Check expiry for unpinned items
	if !it.expiry.IsZero() && time.Now().After(it.expiry) {
		return nil, false
	}
	return it.res, true
}

// Page returns a slice of rows starting at cursor with the given limit.
// Returns the rows, the next cursor position, and whether more rows exist.
func (s *Store) Page(id uuid.UUID, cursor int64, limit int) ([][]string, int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	it, ok := s.items[id]
	if !ok {
		return nil, 0, false
	}
	if !it.expiry.IsZero() && time.Now().After(it.expiry) {
		return nil, 0, false
	}

	rows := it.res.Rows
	total := int64(len(rows))
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		return [][]string{}, total, false
	}

	end := cursor + int64(limit)
	if end > total {
		end = total
	}
	hasMore := end < total
	return rows[cursor:end], end, hasMore
}

// Pin marks a resource as pinned with an optional maxAge override.
func (s *Store) Pin(id uuid.UUID, maxAge *time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	it, ok := s.items[id]
	if !ok {
		return fmt.Errorf("resource not found")
	}

	it.res.Pinned = true
	it.res.MaxAge = maxAge

	if maxAge != nil {
		it.expiry = time.Now().Add(*maxAge)
	} else {
		it.expiry = time.Time{} // never expires
	}
	return nil
}

// Delete removes a resource by ID.
func (s *Store) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("resource not found")
	}
	delete(s.items, id)
	return nil
}

// ─── HTTP Server ─────────────────────────────────────────────────────────────

const defaultPageLimit = 100

type server struct {
	store  *Store
	logger *slog.Logger
}

// Start begins the HTTP data server. Blocks until ctx is cancelled or the
// server fails. Callers should run it as a goroutine.
func Start(ctx context.Context, store *Store, bind string, port int, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("source", "dataserver.Server")

	s := &server{store: store, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /resources", s.handleList)
	mux.HandleFunc("GET /resources/{id}", s.handleGet)
	mux.HandleFunc("POST /resources/{id}/pin", s.handlePin)
	mux.HandleFunc("DELETE /resources/{id}", s.handleDelete)

	addr := net.JoinHostPort(bind, strconv.Itoa(port))
	hs := &http.Server{
		Addr:    addr,
		Handler: recoveryMiddleware(logger, mux),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		hs.Shutdown(shutdownCtx)
	}()

	logger.Info("data server starting", "addr", addr)
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("data server: %w", err)
	}
	return nil
}

func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ─── Handlers ────────────────────────────────────────────────────────────────

func (s *server) handleList(w http.ResponseWriter, r *http.Request) {
	summaries := s.store.List()
	writeJSON(w, http.StatusOK, summaries)
}

func (s *server) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource id")
		return
	}

	res, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}

	cursor := int64(0)
	if c := r.URL.Query().Get("cursor"); c != "" {
		cursor, err = strconv.ParseInt(c, 10, 64)
		if err != nil || cursor < 0 {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
	}

	limit := defaultPageLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		n, parseErr := strconv.Atoi(l)
		if parseErr == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	page, nextCursor, hasMore := s.store.Page(id, cursor, limit)

	resp := map[string]any{
		"resource": map[string]any{
			"id":       res.ID,
			"label":    res.Label,
			"columns":  res.Columns,
			"rowCount": res.RowCount,
		},
		"page": map[string]any{
			"rows":    page,
			"cursor":  nextCursor,
			"hasMore": hasMore,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handlePin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource id")
		return
	}

	var req struct {
		MaxAge *string `json:"maxAge,omitempty"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	var maxAge *time.Duration
	if req.MaxAge != nil {
		d, parseErr := time.ParseDuration(*req.MaxAge)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid maxAge duration")
			return
		}
		maxAge = &d
	}

	if err := s.store.Pin(id, maxAge); err != nil {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "pinned"})
}

func (s *server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource id")
		return
	}

	if err := s.store.Delete(id); err != nil {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
