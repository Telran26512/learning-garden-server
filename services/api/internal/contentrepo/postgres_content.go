package contentrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Telran26512/learning-garden-server/services/api/internal/content"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresContentStore struct {
	db *pgxpool.Pool
}

const contentGraphRelationsSQL = `
	SELECT source_id::text, target_id::text, relation_type
	FROM content_relations
	WHERE source_id = ANY($1::uuid[]) AND target_id = ANY($1::uuid[])
	ORDER BY source_id, target_id, relation_type
`

func NewPostgresContentStore(db *pgxpool.Pool) *PostgresContentStore {
	return &PostgresContentStore{db: db}
}

func (s *PostgresContentStore) Create(ctx context.Context, item content.Item) (content.Item, error) {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return content.Item{}, err
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO content_items (
			id, owner_id, kind, title, slug, summary, body, visibility, status, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text, owner_id::text, kind, title, slug, summary, body, visibility, status, metadata, created_at, updated_at
	`, item.ID, item.OwnerID, item.Kind, item.Title, item.Slug, item.Summary, item.Body, item.Visibility, item.Status, metadata, item.CreatedAt, item.UpdatedAt)
	return scanContentItem(row)
}

func (s *PostgresContentStore) Replace(ctx context.Context, item content.Item) (content.Item, error) {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return content.Item{}, err
	}
	row := s.db.QueryRow(ctx, `
		UPDATE content_items
		SET title = $2,
		    slug = $3,
		    summary = $4,
		    body = $5,
		    visibility = $6,
		    status = $7,
		    metadata = $8,
		    updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, owner_id::text, kind, title, slug, summary, body, visibility, status, metadata, created_at, updated_at
	`, item.ID, item.Title, item.Slug, item.Summary, item.Body, item.Visibility, item.Status, metadata, item.UpdatedAt)
	out, err := scanContentItem(row)
	return out, mapContentStoreError(err)
}

func (s *PostgresContentStore) Delete(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE content_items
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return content.ErrNotFound
	}
	return nil
}

func (s *PostgresContentStore) FindByID(ctx context.Context, id string) (content.Item, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id::text, owner_id::text, kind, title, slug, summary, body, visibility, status, metadata, created_at, updated_at
		FROM content_items
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	item, err := scanContentItem(row)
	return item, mapContentStoreError(err)
}

func (s *PostgresContentStore) List(ctx context.Context, filter content.ListFilter) ([]content.Item, error) {
	where, args := contentFilterSQL(filter)
	limit := ""
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		limit = fmt.Sprintf(" LIMIT $%d", len(args))
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, owner_id::text, kind, title, slug, summary, body, visibility, status, metadata, created_at, updated_at
		FROM content_items
		`+where+`
		ORDER BY updated_at DESC, title ASC
		`+limit, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []content.Item{}
	for rows.Next() {
		item, err := scanContentItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresContentStore) AddRelation(ctx context.Context, relation content.Relation) (content.Relation, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO content_relations (source_id, target_id, relation_type, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (source_id, target_id, relation_type) DO UPDATE SET created_at = EXCLUDED.created_at
		RETURNING source_id::text, target_id::text, relation_type, created_at
	`, relation.SourceID, relation.TargetID, relation.Type, relation.CreatedAt)
	return scanRelation(row)
}

func (s *PostgresContentStore) Backlinks(ctx context.Context, targetID string) ([]content.Backlink, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			r.source_id::text, r.target_id::text, r.relation_type, r.created_at,
			i.id::text, i.owner_id::text, i.kind, i.title, i.slug, i.summary, i.body, i.visibility, i.status, i.metadata, i.created_at, i.updated_at
		FROM content_relations r
		JOIN content_items i ON i.id = r.source_id
		WHERE r.target_id = $1
		  AND i.visibility = 'public'
		  AND i.deleted_at IS NULL
		ORDER BY i.updated_at DESC, i.title ASC
	`, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	backlinks := []content.Backlink{}
	for rows.Next() {
		var relation content.Relation
		var item content.Item
		var metadata []byte
		if err := rows.Scan(
			&relation.SourceID,
			&relation.TargetID,
			&relation.Type,
			&relation.CreatedAt,
			&item.ID,
			&item.OwnerID,
			&item.Kind,
			&item.Title,
			&item.Slug,
			&item.Summary,
			&item.Body,
			&item.Visibility,
			&item.Status,
			&metadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = decodeMetadata(metadata)
		backlinks = append(backlinks, content.Backlink{Relation: relation, Source: item})
	}
	return backlinks, rows.Err()
}

func (s *PostgresContentStore) Graph(ctx context.Context, filter content.ListFilter) (content.Graph, error) {
	items, err := s.List(ctx, filter)
	if err != nil {
		return content.Graph{}, err
	}
	return s.GraphFromItems(ctx, items)
}

func (s *PostgresContentStore) GraphFromItems(ctx context.Context, items []content.Item) (content.Graph, error) {
	nodeIDs := make([]string, 0, len(items))
	nodes := make([]content.GraphNode, 0, len(items))
	for _, item := range items {
		nodeIDs = append(nodeIDs, item.ID)
		nodes = append(nodes, content.GraphNode{
			ID:         item.ID,
			Kind:       item.Kind,
			Title:      item.Title,
			Visibility: item.Visibility,
		})
	}
	if len(nodeIDs) == 0 {
		return content.Graph{Nodes: nodes, Edges: []content.GraphEdge{}}, nil
	}
	rows, err := s.db.Query(ctx, contentGraphRelationsSQL, nodeIDs)
	if err != nil {
		return content.Graph{}, err
	}
	defer rows.Close()
	edges := []content.GraphEdge{}
	for rows.Next() {
		var edge content.GraphEdge
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &edge.Type); err != nil {
			return content.Graph{}, err
		}
		edges = append(edges, edge)
	}
	return content.Graph{Nodes: nodes, Edges: edges}, rows.Err()
}

func contentFilterSQL(filter content.ListFilter) (string, []any) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	if filter.Kind != "" {
		args = append(args, filter.Kind)
		conditions = append(conditions, fmt.Sprintf("kind = $%d", len(args)))
	}
	if filter.OwnerID != "" {
		args = append(args, filter.OwnerID)
		conditions = append(conditions, fmt.Sprintf("owner_id = $%d", len(args)))
	}
	if filter.Visibility != "" {
		args = append(args, filter.Visibility)
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", len(args)))
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

type contentRow interface {
	Scan(dest ...any) error
}

func scanContentItem(row contentRow) (content.Item, error) {
	var item content.Item
	var metadata []byte
	if err := row.Scan(
		&item.ID,
		&item.OwnerID,
		&item.Kind,
		&item.Title,
		&item.Slug,
		&item.Summary,
		&item.Body,
		&item.Visibility,
		&item.Status,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return content.Item{}, mapContentStoreError(err)
	}
	item.Metadata = decodeMetadata(metadata)
	return item, nil
}

func scanRelation(row contentRow) (content.Relation, error) {
	var relation content.Relation
	if err := row.Scan(&relation.SourceID, &relation.TargetID, &relation.Type, &relation.CreatedAt); err != nil {
		return content.Relation{}, mapContentStoreError(err)
	}
	return relation, nil
}

func decodeMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func mapContentStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return content.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return content.ErrConflict
		case "23503":
			return content.ErrNotFound
		}
	}
	return err
}
