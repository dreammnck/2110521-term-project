CREATE TABLE IF NOT EXISTS parcels (
  id                UUID PRIMARY KEY,
  room_id           UUID NOT NULL,
  recipient_user_id UUID,
  carrier           TEXT NOT NULL DEFAULT '',
  tracking_no       TEXT NOT NULL DEFAULT '',
  note              TEXT NOT NULL DEFAULT '',
  photo_path        TEXT NOT NULL DEFAULT '',
  status            TEXT NOT NULL DEFAULT 'WAITING' CHECK (status IN ('WAITING','PICKED_UP')),
  registered_by     UUID NOT NULL,
  received_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  picked_up_at      TIMESTAMPTZ,
  picked_up_by      UUID
);
CREATE INDEX IF NOT EXISTS parcels_room_idx ON parcels(room_id, status);
