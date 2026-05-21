package content

import (
	"context"
	"testing"
	"time"
)

func TestPortfolioBuildsGraphFromAlreadyListedItems(t *testing.T) {
	store := &queryCountingStore{
		items: []Item{
			{
				ID:         "track-1",
				OwnerID:    "user-1",
				Kind:       KindTrack,
				Title:      "Track",
				Slug:       "track",
				Visibility: VisibilityPublic,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			{
				ID:         "note-1",
				OwnerID:    "user-1",
				Kind:       KindNote,
				Title:      "Note",
				Slug:       "note",
				Visibility: VisibilityPublic,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		},
	}
	service := NewService(store)

	portfolio, err := service.Portfolio(context.Background(), PublicProfile{ID: "user-1", DisplayName: "Ada"}, 20)
	if err != nil {
		t.Fatalf("Portfolio returned error: %v", err)
	}

	if store.listCalls != 1 {
		t.Fatalf("List calls = %d, want 1", store.listCalls)
	}
	if store.graphCalls != 0 {
		t.Fatalf("Graph calls = %d, want 0", store.graphCalls)
	}
	if store.graphFromItemsCalls != 1 {
		t.Fatalf("GraphFromItems calls = %d, want 1", store.graphFromItemsCalls)
	}
	if store.graphFromItemsLen != 2 {
		t.Fatalf("GraphFromItems received %d items, want 2", store.graphFromItemsLen)
	}
	if len(portfolio.Graph.Nodes) != 2 {
		t.Fatalf("Portfolio graph nodes = %d, want 2", len(portfolio.Graph.Nodes))
	}
}

type queryCountingStore struct {
	items               []Item
	listCalls           int
	graphCalls          int
	graphFromItemsCalls int
	graphFromItemsLen   int
}

func (s *queryCountingStore) Create(context.Context, Item) (Item, error) {
	panic("not used")
}

func (s *queryCountingStore) Replace(context.Context, Item) (Item, error) {
	panic("not used")
}

func (s *queryCountingStore) Delete(context.Context, string) error {
	panic("not used")
}

func (s *queryCountingStore) FindByID(context.Context, string) (Item, error) {
	panic("not used")
}

func (s *queryCountingStore) List(context.Context, ListFilter) ([]Item, error) {
	s.listCalls++
	return append([]Item(nil), s.items...), nil
}

func (s *queryCountingStore) AddRelation(context.Context, Relation) (Relation, error) {
	panic("not used")
}

func (s *queryCountingStore) Backlinks(context.Context, string) ([]Backlink, error) {
	panic("not used")
}

func (s *queryCountingStore) Graph(context.Context, ListFilter) (Graph, error) {
	s.graphCalls++
	return Graph{}, nil
}

func (s *queryCountingStore) GraphFromItems(_ context.Context, items []Item) (Graph, error) {
	s.graphFromItemsCalls++
	s.graphFromItemsLen = len(items)
	nodes := make([]GraphNode, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, GraphNode{
			ID:         item.ID,
			Kind:       item.Kind,
			Title:      item.Title,
			Visibility: item.Visibility,
		})
	}
	return Graph{Nodes: nodes, Edges: []GraphEdge{}}, nil
}
