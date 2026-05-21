package content

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

type Store interface {
	Create(context.Context, Item) (Item, error)
	Replace(context.Context, Item) (Item, error)
	Delete(context.Context, string) error
	FindByID(context.Context, string) (Item, error)
	List(context.Context, ListFilter) ([]Item, error)
	AddRelation(context.Context, Relation) (Relation, error)
	Backlinks(context.Context, string) ([]Backlink, error)
	Graph(context.Context, ListFilter) (Graph, error)
	GraphFromItems(context.Context, []Item) (Graph, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, ownerID string, input CreateItemInput) (Item, error) {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return Item{}, ErrForbidden
	}
	item := Item{
		ID:         newID(),
		OwnerID:    ownerID,
		Kind:       input.Kind,
		Title:      strings.TrimSpace(input.Title),
		Slug:       normalizeSlug(input.Slug),
		Summary:    strings.TrimSpace(input.Summary),
		Body:       strings.TrimSpace(input.Body),
		Visibility: input.Visibility,
		Status:     strings.TrimSpace(input.Status),
		Metadata:   cloneMetadata(input.Metadata),
		CreatedAt:  s.now(),
		UpdatedAt:  s.now(),
	}
	if item.Slug == "" {
		item.Slug = normalizeSlug(item.Title)
	}
	if item.Visibility == "" {
		item.Visibility = VisibilityPrivate
	}
	if item.Status == "" {
		item.Status = "draft"
	}
	if err := validateItem(item); err != nil {
		return Item{}, err
	}
	return s.store.Create(ctx, item)
}

func (s *Service) ListPublic(ctx context.Context, filter ListFilter) ([]Item, error) {
	filter.Visibility = VisibilityPublic
	return s.store.List(ctx, filter)
}

func (s *Service) ListVisible(ctx context.Context, filter ListFilter, viewerID string) ([]Item, error) {
	if strings.TrimSpace(viewerID) == "" {
		return s.ListPublic(ctx, filter)
	}
	return s.store.List(ctx, filter)
}

func (s *Service) FindVisible(ctx context.Context, id string, viewerID string) (Item, error) {
	item, err := s.store.FindByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return Item{}, err
	}
	if canView(item, viewerID) {
		return item, nil
	}
	return Item{}, ErrNotFound
}

func (s *Service) Update(ctx context.Context, id string, actorID string, input UpdateItemInput) (Item, error) {
	item, err := s.store.FindByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return Item{}, err
	}
	if item.OwnerID != strings.TrimSpace(actorID) {
		return Item{}, ErrForbidden
	}
	if input.Title != nil {
		item.Title = strings.TrimSpace(*input.Title)
	}
	if input.Slug != nil {
		item.Slug = normalizeSlug(*input.Slug)
	}
	if input.Summary != nil {
		item.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Body != nil {
		item.Body = strings.TrimSpace(*input.Body)
	}
	if input.Visibility != nil {
		item.Visibility = *input.Visibility
	}
	if input.Status != nil {
		item.Status = strings.TrimSpace(*input.Status)
	}
	if input.Metadata != nil {
		item.Metadata = cloneMetadata(*input.Metadata)
	}
	if item.Slug == "" {
		item.Slug = normalizeSlug(item.Title)
	}
	item.UpdatedAt = s.now()
	if err := validateItem(item); err != nil {
		return Item{}, err
	}
	return s.store.Replace(ctx, item)
}

func (s *Service) Delete(ctx context.Context, id string, actorID string) error {
	item, err := s.store.FindByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if item.OwnerID != strings.TrimSpace(actorID) {
		return ErrForbidden
	}
	return s.store.Delete(ctx, item.ID)
}

func (s *Service) AddRelation(ctx context.Context, sourceID string, actorID string, targetID string, relationType string) (Relation, error) {
	source, err := s.store.FindByID(ctx, strings.TrimSpace(sourceID))
	if err != nil {
		return Relation{}, err
	}
	if source.OwnerID != strings.TrimSpace(actorID) {
		return Relation{}, ErrForbidden
	}
	target, err := s.FindVisible(ctx, targetID, actorID)
	if err != nil {
		return Relation{}, err
	}
	relation := Relation{
		SourceID:  source.ID,
		TargetID:  target.ID,
		Type:      normalizeRelationType(relationType),
		CreatedAt: s.now(),
	}
	if relation.Type == "" {
		return Relation{}, ErrInvalidInput
	}
	return s.store.AddRelation(ctx, relation)
}

func (s *Service) Backlinks(ctx context.Context, targetID string) ([]Backlink, error) {
	target, err := s.FindVisible(ctx, targetID, "")
	if err != nil {
		return nil, err
	}
	return s.store.Backlinks(ctx, target.ID)
}

func (s *Service) Graph(ctx context.Context, filter ListFilter) (Graph, error) {
	filter.Visibility = VisibilityPublic
	items, err := s.store.List(ctx, filter)
	if err != nil {
		return Graph{}, err
	}
	return s.store.GraphFromItems(ctx, items)
}

func (s *Service) Stats(ctx context.Context, ownerID string) (Stats, error) {
	items, err := s.ListPublic(ctx, ListFilter{OwnerID: ownerID})
	if err != nil {
		return Stats{}, err
	}
	return statsForItems(items), nil
}

func (s *Service) Portfolio(ctx context.Context, profile PublicProfile, limit int) (Portfolio, error) {
	filter := ListFilter{OwnerID: profile.ID, Limit: limit}
	items, err := s.ListPublic(ctx, filter)
	if err != nil {
		return Portfolio{}, err
	}
	graph, err := s.store.GraphFromItems(ctx, items)
	if err != nil {
		return Portfolio{}, err
	}
	grouped := map[Kind][]Item{
		KindTrack:      {},
		KindNote:       {},
		KindPaper:      {},
		KindExperiment: {},
	}
	for _, item := range items {
		grouped[item.Kind] = append(grouped[item.Kind], item)
	}
	stats := profile.Stats
	if stats == (Stats{}) {
		stats = statsForItems(items)
	}
	profile.Stats = stats
	return Portfolio{
		Profile: profile,
		Items:   grouped,
		Graph:   graph,
		Stats:   stats,
		Topics:  topicsForItems(items),
		Recent:  feedRowsForItems(profile.DisplayName, items, 8),
	}, nil
}

func (s *Service) CommunityFeed(ctx context.Context, limit int) ([]CommunityFeedRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	items, err := s.ListPublic(ctx, ListFilter{Limit: limit})
	if err != nil {
		return nil, err
	}
	return feedRowsForItems("", items, limit), nil
}

func canView(item Item, viewerID string) bool {
	if item.Visibility == VisibilityPublic || item.Visibility == VisibilityUnlisted {
		return true
	}
	return item.OwnerID == strings.TrimSpace(viewerID) && viewerID != ""
}

func validateItem(item Item) error {
	if !validKind(item.Kind) || !validVisibility(item.Visibility) {
		return ErrInvalidInput
	}
	if strings.TrimSpace(item.OwnerID) == "" || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Slug) == "" {
		return ErrInvalidInput
	}
	return nil
}

func validKind(kind Kind) bool {
	switch kind {
	case KindTrack, KindNote, KindPaper, KindExperiment:
		return true
	default:
		return false
	}
}

func validVisibility(visibility Visibility) bool {
	switch visibility {
	case VisibilityPrivate, VisibilityPublic, VisibilityUnlisted:
		return true
	default:
		return false
	}
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-")
	value = replacer.Replace(value)
	return strings.Trim(value, "-")
}

func normalizeRelationType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func cloneMetadata(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func statsForItems(items []Item) Stats {
	var stats Stats
	for _, item := range items {
		switch item.Kind {
		case KindTrack:
			stats.Tracks++
		case KindNote:
			stats.Notes++
		case KindPaper:
			stats.Papers++
		case KindExperiment:
			stats.Experiments++
		}
	}
	return stats
}

func topicsForItems(items []Item) map[string]int {
	topics := map[string]int{}
	for _, item := range items {
		raw, ok := item.Metadata["tags"].([]any)
		if !ok {
			continue
		}
		for _, value := range raw {
			tag, ok := value.(string)
			if !ok {
				continue
			}
			tag = strings.TrimSpace(tag)
			if tag != "" {
				topics[tag]++
			}
		}
	}
	return topics
}

func feedRowsForItems(actor string, items []Item, limit int) []CommunityFeedRow {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	rows := make([]CommunityFeedRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, CommunityFeedRow{
			ID:        item.ID,
			Type:      "content_published",
			Actor:     actor,
			Title:     item.Title,
			Summary:   item.Summary,
			Kind:      item.Kind,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			Metadata:  cloneMetadata(item.Metadata),
		})
	}
	return rows
}

func newID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return strings.Join([]string{
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	}, "-")
}
