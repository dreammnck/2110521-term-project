CREATE TABLE IF NOT EXISTS buildings (
  id   UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS rooms (
  id          UUID PRIMARY KEY,
  building_id UUID NOT NULL REFERENCES buildings(id),
  floor       INT  NOT NULL,
  number      TEXT NOT NULL,
  UNIQUE (building_id, number)
);
CREATE TABLE IF NOT EXISTS users (
  id            UUID PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name          TEXT NOT NULL,
  role          TEXT NOT NULL CHECK (role IN ('staff','resident')),
  room_id       UUID REFERENCES rooms(id)
);
CREATE INDEX IF NOT EXISTS users_room_idx ON users(room_id);
