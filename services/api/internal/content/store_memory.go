package content

import (
	"context"
	"sort"
	"sync"
)

type MemoryStore struct {
	mu        sync.Mutex
	items     map[string]Item
	relations map[string]Relation
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items:     map[string]Item{},
		relations: map[string]Relation{},
	}
}

func (s *MemoryStore) Create(_ context.Context, item Item) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.OwnerID == item.OwnerID && existing.Kind == item.Kind && existing.Slug == item.Slug {
			return Item{}, ErrConflict
		}
	}
	s.items[item.ID] = cloneItem(item)
	return cloneItem(item), nil
}

func (s *MemoryStore) Replace(_ context.Context, item Item) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; !ok {
		return Item{}, ErrNotFound
	}
	for _, existing := range s.items {
		if existing.ID != item.ID && existing.OwnerID == item.OwnerID && existing.Kind == item.Kind && existing.Slug == item.Slug {
			return Item{}, ErrConflict
		}
	}
	s.items[item.ID] = cloneItem(item)
	return cloneItem(item), nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	for key, relation := range s.relations {
		if relation.SourceID == id || relation.TargetID == id {
			delete(s.relations, key)
		}
	}
	return nil
}

func (s *MemoryStore) FindByID(_ context.Context, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return cloneItem(item), nil
}

func (s *MemoryStore) List(_ context.Context, filter ListFilter) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if !matchesFilter(item, filter) {
			continue
		}
		items = append(items, cloneItem(item))
	}
	sortItems(items)
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (s *MemoryStore) AddRelation(_ context.Context, relation Relation) (Relation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[relation.SourceID]; !ok {
		return Relation{}, ErrNotFound
	}
	if _, ok := s.items[relation.TargetID]; !ok {
		return Relation{}, ErrNotFound
	}
	s.relations[relationKey(relation)] = relation
	return relation, nil
}

func (s *MemoryStore) Backlinks(_ context.Context, targetID string) ([]Backlink, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	backlinks := []Backlink{}
	for _, relation := range s.relations {
		if relation.TargetID != targetID {
			continue
		}
		source, ok := s.items[relation.SourceID]
		if !ok || source.Visibility != VisibilityPublic {
			continue
		}
		backlinks = append(backlinks, Backlink{
			Relation: relation,
			Source:   cloneItem(source),
		})
	}
	sort.SliceStable(backlinks, func(i, j int) bool {
		return backlinks[i].Source.UpdatedAt.After(backlinks[j].Source.UpdatedAt)
	})
	return backlinks, nil
}

func (s *MemoryStore) Graph(_ context.Context, filter ListFilter) (Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []Item{}
	for _, item := range s.items {
		if !matchesFilter(item, filter) {
			continue
		}
		items = append(items, cloneItem(item))
	}
	sortItems(items)
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return s.graphFromItemsLocked(items), nil
}

func (s *MemoryStore) GraphFromItems(_ context.Context, items []Item) (Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.graphFromItemsLocked(items), nil
}

func (s *MemoryStore) graphFromItemsLocked(items []Item) Graph {
	nodes := []GraphNode{}
	for _, item := range items {
		nodes = append(nodes, GraphNode{
			ID:         item.ID,
			Kind:       item.Kind,
			Title:      item.Title,
			Visibility: item.Visibility,
		})
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Title < nodes[j].Title })
	nodeIDs := map[string]struct{}{}
	for _, node := range nodes {
		nodeIDs[node.ID] = struct{}{}
	}
	edges := []GraphEdge{}
	for _, relation := range s.relations {
		if _, ok := nodeIDs[relation.SourceID]; !ok {
			continue
		}
		if _, ok := nodeIDs[relation.TargetID]; !ok {
			continue
		}
		edges = append(edges, GraphEdge{
			SourceID: relation.SourceID,
			TargetID: relation.TargetID,
			Type:     relation.Type,
		})
	}
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].SourceID == edges[j].SourceID {
			return edges[i].TargetID < edges[j].TargetID
		}
		return edges[i].SourceID < edges[j].SourceID
	})
	return Graph{Nodes: nodes, Edges: edges}
}

func matchesFilter(item Item, filter ListFilter) bool {
	if filter.Kind != "" && item.Kind != filter.Kind {
		return false
	}
	if filter.OwnerID != "" && item.OwnerID != filter.OwnerID {
		return false
	}
	if filter.Visibility != "" && item.Visibility != filter.Visibility {
		return false
	}
	return true
}

func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].Title < items[j].Title
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
}

func cloneItem(item Item) Item {
	item.Metadata = cloneMetadata(item.Metadata)
	return item
}

func relationKey(relation Relation) string {
	return relation.SourceID + "\x00" + relation.TargetID + "\x00" + relation.Type
}
