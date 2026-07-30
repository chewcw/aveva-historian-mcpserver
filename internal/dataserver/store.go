package dataserver

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Field struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type Row struct {
	ID    int64    `json:"id"`
	Cells []string `json:"cells"`
}

type Resource struct {
	ID        uuid.UUID      `json:"id"`
	Label     string         `json:"label"`
	MimeType  string         `json:"mimeType"`
	CreatedAt time.Time      `json:"createdAt"`
	TTL       time.Duration  `json:"ttl"`
	Pinned    bool           `json:"pinned"`
	MaxAge    *time.Duration `json:"maxAge,omitempty"`
	Columns   []Field        `json:"columns"`
	Rows      []Row          `json:"-"`
	RowCount  int            `json:"rowCount"`
}

type ResourceSummary struct {
	ID        uuid.UUID     `json:"id"`
	Label     string        `json:"label"`
	RowCount  int           `json:"rowCount"`
	CreatedAt time.Time     `json:"createdAt"`
	TTL       time.Duration `json:"ttl"`
	Pinned    bool          `json:"pinned"`
	ExpiresAt time.Time     `json:"expiresAt"`
}

type Store struct {
	mu    sync.RWMutex
	items map[uuid.UUID]*Resource
	ttl   time.Duration
	seq   int64 // global sequence for row IDs
	clock func() time.Time
}

func NewStore(defaultTTL time.Duration) *Store {
	return &Store{
		items: make(map[uuid.UUID]*Resource),
		ttl:   defaultTTL,
		seq:   0,
		clock: time.Now,
	}
}

func (s *Store) Put(label, mimeType string, columns []Field, rows [][]string) (*Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New()
	now := s.clock()
	stored := make([]Row, len(rows))
	for i, r := range rows {
		s.seq++
		stored[i] = Row{ID: s.seq, Cells: r}
	}

	cols := make([]Field, len(columns))
	copy(cols, columns)

	resource := &Resource{
		ID:        id,
		Label:     label,
		MimeType:  mimeType,
		CreatedAt: now,
		TTL:       s.ttl,
		Columns:   cols,
		Rows:      stored,
		RowCount:  len(stored),
	}
	s.items[id] = resource
	return resource, nil
}

func (s *Store) Get(id uuid.UUID) (*Resource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.items[id]
	return r, ok
}

func (s *Store) Page(id uuid.UUID, cursor int64, limit int) ([]Row, int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.items[id]
	if !ok {
		return nil, 0, false
	}

	start := sort.Search(len(resource.Rows), func(i int) bool {
		return resource.Rows[i].ID > cursor
	})
	if start >= len(resource.Rows) {
		return nil, 0, false
	}

	end := start + limit
	if end > len(resource.Rows) {
		end = len(resource.Rows)
	}

	page := make([]Row, end-start)
	copy(page, resource.Rows[start:end])

	lastID := int64(0)
	if len(page) > 0 {
		lastID = page[len(page)-1].ID
	}
	hasMore := end < len(resource.Rows)

	return page, lastID, hasMore
}

// List returns the ResourceSummary ordered by CreatedAt.
func (s *Store) List() []ResourceSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ResourceSummary, 0, len(s.items))
	for _, r := range s.items {
		out = append(out, ResourceSummary{
			ID:        r.ID,
			Label:     r.Label,
			RowCount:  r.RowCount,
			CreatedAt: r.CreatedAt,
			TTL:       r.TTL,
			Pinned:    r.Pinned,
			ExpiresAt: r.CreatedAt.Add(r.TTL),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (s *Store) Pin(id uuid.UUID, maxAge *time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.items[id]
	if !ok {
		return fmt.Errorf("resource not found")
	}
	r.Pinned = true
	r.MaxAge = maxAge
	return nil
}

func (s *Store) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("resource not found")
	}
	delete(s.items, id)
	return nil
}

func (s *Store) Sweep(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var removed int
	for id, r := range s.items {
		if r.Pinned {
			if r.MaxAge != nil && now.After(r.CreatedAt.Add(*r.MaxAge)) {
				delete(s.items, id)
				removed++
			}
			continue
		}
		if now.After(r.CreatedAt.Add(r.TTL)) {
			delete(s.items, id)
			removed++
		}
	}
	return removed
}
