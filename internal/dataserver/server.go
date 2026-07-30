package dataserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

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
