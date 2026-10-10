CREATE TABLE IF NOT EXISTS announcements (
  id                 UUID PRIMARY KEY,
  title              TEXT NOT NULL,
  body               TEXT NOT NULL,
  category           TEXT NOT NULL DEFAULT 'GENERAL',
  status             TEXT NOT NULL DEFAULT 'PUBLISHED' CHECK (status IN ('DRAFT','PUBLISHED','ARCHIVED')),
  target_type        TEXT NOT NULL DEFAULT 'ALL' CHECK (target_type IN ('ALL','BUILDING','FLOOR')),
  target_building_id TEXT NOT NULL DEFAULT '',
  target_floor       INT  NOT NULL DEFAULT 0,
  publish_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         TEXT NOT NULL DEFAULT '',
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS announcement_reads (
  announcement_id UUID NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
  user_id         TEXT NOT NULL,
  read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (announcement_id, user_id)
);
