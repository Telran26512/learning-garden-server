package content

import (
	"errors"
	"time"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Kind string

const (
	KindTrack      Kind = "track"
	KindNote       Kind = "note"
	KindPaper      Kind = "paper"
	KindExperiment Kind = "experiment"
)

type Visibility string

const (
	VisibilityPrivate  Visibility = "private"
	VisibilityPublic   Visibility = "public"
	VisibilityUnlisted Visibility = "unlisted"
)

type Item struct {
	ID         string         `json:"id"`
	OwnerID    string         `json:"ownerId"`
	Kind       Kind           `json:"kind"`
	Title      string         `json:"title"`
	Slug       string         `json:"slug"`
	Summary    string         `json:"summary"`
	Body       string         `json:"body"`
	Visibility Visibility     `json:"visibility"`
	Status     string         `json:"status"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type CreateItemInput struct {
	Kind       Kind           `json:"kind"`
	Title      string         `json:"title"`
	Slug       string         `json:"slug"`
	Summary    string         `json:"summary"`
	Body       string         `json:"body"`
	Visibility Visibility     `json:"visibility"`
	Status     string         `json:"status"`
	Metadata   map[string]any `json:"metadata"`
}

type UpdateItemInput struct {
	Title      *string         `json:"title"`
	Slug       *string         `json:"slug"`
	Summary    *string         `json:"summary"`
	Body       *string         `json:"body"`
	Visibility *Visibility     `json:"visibility"`
	Status     *string         `json:"status"`
	Metadata   *map[string]any `json:"metadata"`
}

type ListFilter struct {
	Kind       Kind
	OwnerID    string
	Visibility Visibility
	Limit      int
}

type Relation struct {
	SourceID  string    `json:"sourceId"`
	TargetID  string    `json:"targetId"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
}

type Backlink struct {
	Relation Relation `json:"relation"`
	Source   Item     `json:"source"`
}

type GraphNode struct {
	ID         string     `json:"id"`
	Kind       Kind       `json:"kind"`
	Title      string     `json:"title"`
	Visibility Visibility `json:"visibility"`
}

type GraphEdge struct {
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
}

type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type Stats struct {
	Tracks      int `json:"tracks"`
	Notes       int `json:"notes"`
	Papers      int `json:"papers"`
	Experiments int `json:"experiments"`
}

type PublicProfile struct {
	ID          string    `json:"id"`
	Handle      string    `json:"handle"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"createdAt"`
	Stats       Stats     `json:"stats"`
}

type Portfolio struct {
	Profile PublicProfile      `json:"profile"`
	Items   map[Kind][]Item    `json:"items"`
	Graph   Graph              `json:"graph"`
	Stats   Stats              `json:"stats"`
	Topics  map[string]int     `json:"topics"`
	Recent  []CommunityFeedRow `json:"recent"`
}

type CommunityFeedRow struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Actor     string         `json:"actor"`
	Title     string         `json:"title"`
	Summary   string         `json:"summary"`
	Kind      Kind           `json:"kind"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Metadata  map[string]any `json:"metadata"`
}
