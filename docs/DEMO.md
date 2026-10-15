# Demo video script (target ≈ 4:30, hard limit 5:00)

ไทย: [DEMO.th.md](DEMO.th.md)

**Must show:** all 3 business use cases + authentication · 2 database types (**PostgreSQL = RDBMS**, **MongoDB = NoSQL**) · a service using a message broker (**RabbitMQ**).

## Before you hit record (5 min)

```bash
make clean && make up          # fresh data, ~1 min after images are built
docker compose ps              # all containers "Up"/healthy
```
Optional sanity run: `make smoke`, then `docker compose down -v && docker compose up -d` again so the recording starts with clean data.

Set up the screen (one display, 4 things visible by switching, no scrolling needed):

| Window | URL / command | Notes |
|---|---|---|
| **A. Staff app** (desktop width) | http://localhost:3000 (3100 if you use the local `.env`) | not signed in yet |
| **B. Resident app** (phone size 390×844, use DevTools device mode or resize the window) | http://localhost:3001 (3101) | not signed in yet |
| **C. RabbitMQ UI** | http://localhost:15672 → login `guest`/`guest` → **Queues → notification-adapter** | leave this tab open |
| **D. Terminal** (large font) | project folder; run the DB commands below | keep two panes: one for `psql`/`mongosh`, one for logs |

Have a small JPG ready for the parcel photo (`~/Desktop/parcel.jpg`).
Start a log tail in the second terminal pane so it fills during the demo:
```bash
docker compose logs -f notification-adapter | grep --line-buffered "provider: send"
```

Accounts (password `password123`): `staff@condo.local`, `resident1@condo.local` (Room A101, Building A), `resident7@condo.local` (Building B).

---

## Script

### 0:00 – 0:25 · Intro (show README architecture diagram)
> "This is a condominium management system: Go microservices, Next.js frontends. Residents and juristic staff use it for parcels, utility bills and announcements. Every service owns its data: PostgreSQL for relational data, MongoDB for notifications, and RabbitMQ carries events between services."

Point at the diagram: gateway → services → **PostgreSQL** (identity, parcel, announcement, billing) / **MongoDB** (notification) / **RabbitMQ**.

### 0:25 – 0:55 · Authentication
1. **Window A (staff):** try `resident1@condo.local` → red error *"This app is for juristic staff only"*. 
   > "Role is enforced: a resident can't enter the staff app."
2. Sign in as `staff@condo.local` → dashboard.
3. **Window B (resident):** sign in as `resident1@condo.local` → Home.
   > "Login goes through the Identity service, issues a JWT; the gateway verifies it and passes identity downstream."

### 0:55 – 1:50 · Use case 1: Parcel management (+ RabbitMQ)
1. **Window C:** show queue `notification-adapter`: 1 consumer, 0 messages. 
2. **Window A → Parcels:** Building A, room **A101**, carrier *Kerry*, tracking `TH123`, attach photo → **Register parcel**.
3. **Window C:** the message-rate graph blips (published/acked) → 
   > "Parcel service published a `ParcelReceived` event to RabbitMQ; the Notification adapter consumed it."
4. **Terminal log pane:** shows `notification provider: send … New parcel arrived`.
5. **Window B:** bell badge appears (pull to refresh or wait ≤15 s) → open **Parcels** → photo + *Waiting* → **Confirm pickup** → sheet → *Yes, I have it*.
6. **Window A:** parcel moves to *Picked up*; dashboard count drops. 
   > "`ParcelAccepted` event → staff notified."

### 1:50 – 2:55 · Use case 2: Utility billing & payment
1. **Window A → Utility bills → Single room:** A101, cycle = this month, water `10`, electricity `100`, due in 14 days → **Issue bill** (shows ฿880.00).
2. **Window B → Bills:** new bill ฿880 *Unpaid* (notification arrived too) → open it → **Pay ฿880.00** → *Continue to payment*.
3. Mock gateway page opens → click **Pay now**.
   > "The payment gateway is an external system. It calls our Payment adapter with an HMAC-signed webhook; the adapter verifies it before Billing marks the bill paid."
4. Redirected to the app: *Payment successful* → **View receipt** (receipt no., reference).
5. **Window A → Utility bills:** same bill now *Paid*.

### 2:55 – 3:30 · Use case 3: Announcements
1. **Window A → Announcements → New:** title *Lift maintenance*, audience **A specific building → Building A** → **Publish**.
2. **Window B → News:** announcement appears with *New* badge → open it (marks read).
3. Quick contrast (terminal, 10 s) — a Building B resident doesn't see it:
   ```bash
   T=$(curl -s -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"resident7@condo.local","password":"password123"}' | jq -r .token)
   curl -s localhost:8080/api/v1/announcements -H "Authorization: Bearer $T"      # → []
   ```
   > "Targeting is enforced in the Announcement service, which the gateway reaches over gRPC."

### 3:30 – 4:30 · Databases + broker evidence (terminal)
**RDBMS – PostgreSQL** (two quick queries):
```bash
docker compose exec postgres psql -U condo -d parcel_db  -c "select id, carrier, status, received_at, picked_up_at from parcels order by received_at desc limit 3;"
docker compose exec postgres psql -U condo -d billing_db -c "select b.cycle, b.total, b.status, p.payment_reference, p.payment_status from utility_bills b join payments p on p.bill_id=b.id order by p.created_at desc limit 3;"
```
> "Relational data with foreign keys: a payment must reference an existing bill."

**NoSQL – MongoDB** (notifications):
```bash
docker compose exec mongo mongosh notification_db --quiet --eval 'db.notifications.find({}, {title:1, type:1, read_at:1, created_at:1}).sort({created_at:-1}).limit(5).forEach(printjson); printjson(db.notifications.getIndexes().map(i => i.name))'
```
> "Notifications are documents in MongoDB. A unique index on event id + user makes event handling idempotent — redelivered events can't create duplicates."

**RabbitMQ:** switch to Window C → **Exchanges → `condo.events`** (topic) and **Queues → `notification-adapter`** → show *Total / Ack* counters.
> "Four services publish events — parcel, announcement, billing — and the Notification adapter consumes them."

### 4:30 – 4:50 · Wrap-up
> "Three use cases, JWT authentication with roles, PostgreSQL plus MongoDB, and RabbitMQ for event-driven notifications — all matching our ADRs: microservices, REST plus gRPC, event-driven notification, relational data, and an external payment gateway."

---

## Timing tips
- Rehearse once; the slowest parts are the photo upload and the payment redirect — keep the photo small.
- If a notification badge is slow, **pull down to refresh** in the resident window (or switch tabs; data revalidates on focus).
- Cut dead air in editing rather than speeding up the narration.

## If something breaks on the day
| Symptom | Fix |
|---|---|
| Page won't load | `docker compose ps`; `docker compose up -d` |
| Port 3000/3001 already used | put `STAFF_PORT=3100` / `RESIDENT_PORT=3101` in `.env`, `docker compose up -d` |
| No notifications | `docker compose logs notification-adapter` (needs RabbitMQ + Mongo healthy) |
| `rabbitmq` exits right after a clean start (`.erlang.cookie: eacces` in `docker compose logs rabbitmq`) | known first-start flake; just run `docker compose up -d` again |
| Want a clean slate | `make clean && make up` |
