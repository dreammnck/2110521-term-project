CREATE TABLE IF NOT EXISTS utility_bills (
  id              UUID PRIMARY KEY,
  room_id         UUID NOT NULL,
  cycle           TEXT NOT NULL,
  water_units     NUMERIC(10,2) NOT NULL DEFAULT 0,
  water_amount    NUMERIC(12,2) NOT NULL DEFAULT 0,
  electric_units  NUMERIC(10,2) NOT NULL DEFAULT 0,
  electric_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
  total           NUMERIC(12,2) NOT NULL,
  due_date        DATE NOT NULL,
  status          TEXT NOT NULL DEFAULT 'UNPAID' CHECK (status IN ('UNPAID','PENDING','PAID')),
  issued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (room_id, cycle)
);
CREATE INDEX IF NOT EXISTS bills_room_idx ON utility_bills(room_id, cycle DESC);
CREATE TABLE IF NOT EXISTS payments (
  id                UUID PRIMARY KEY,
  bill_id           UUID NOT NULL REFERENCES utility_bills(id),
  payment_reference TEXT NOT NULL UNIQUE,
  amount            NUMERIC(12,2) NOT NULL,
  payment_status    TEXT NOT NULL DEFAULT 'PENDING' CHECK (payment_status IN ('PENDING','SUCCEEDED','FAILED')),
  paid_at           TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS payments_bill_idx ON payments(bill_id);
