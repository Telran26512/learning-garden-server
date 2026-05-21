CREATE TABLE IF NOT EXISTS content_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('track', 'note', 'paper', 'experiment')),
  title text NOT NULL,
  slug text NOT NULL,
  summary text NOT NULL DEFAULT '',
  body text NOT NULL DEFAULT '',
  visibility text NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public', 'unlisted')),
  status text NOT NULL DEFAULT 'draft',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  UNIQUE (owner_id, kind, slug)
);

CREATE INDEX IF NOT EXISTS content_items_owner_idx ON content_items (owner_id);
CREATE INDEX IF NOT EXISTS content_items_public_idx ON content_items (visibility, kind, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS content_items_metadata_gin_idx ON content_items USING gin (metadata);

CREATE TABLE IF NOT EXISTS content_relations (
  source_id uuid NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
  target_id uuid NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
  relation_type text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (source_id, target_id, relation_type)
);

CREATE INDEX IF NOT EXISTS content_relations_target_idx ON content_relations (target_id);
CREATE INDEX IF NOT EXISTS content_relations_source_idx ON content_relations (source_id);
