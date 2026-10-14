# ADR 06: Use MongoDB (NoSQL) for notification storage

## Context
ADR-04 chose a relational database for the core data (users, rooms, parcels, announcements, bills, payments), where
relationships and transactional integrity matter. Notifications are different: they are produced by events from several
services (`ParcelReceived`, `ParcelAccepted`, `AnnouncementPublished`, `UtilityBillIssued`, `PaymentCompleted`), one
announcement fans out to every targeted resident, the payload differs per event type and will grow (e.g. push, LINE),
and they are only ever read back by their owner as a recent-first inbox. ADR-01 already notes that the Notification
service has the highest and spikiest write volume.

## Decision
The Notification Adapter stores notifications in **MongoDB** (`notification_db.notifications`), one document per
(event, recipient). All other services keep PostgreSQL.

- Unique index on `(event_id, user_id)` makes event consumption idempotent (RabbitMQ redelivery cannot duplicate).
- Index on `(user_id, created_at desc)` serves the inbox query.
- No cross-service foreign keys: documents hold plain user/reference ids (`ref_type`, `ref_id`).

## Status
Accepted

## Consequences
**Positive**
- Flexible per-event content without schema migrations; easy to add channels or fields later.
- Cheap fan-out writes; the notification workload scales independently of the relational services.
- Keeps the relational databases focused on data that needs integrity.

**Trade-offs**
- Two database technologies to run, back up and monitor.
- No joins or multi-document transactions; acceptable because notifications are append-mostly and self-contained.
- Eventual consistency between a business change and its notification (already inherent in ADR-03).
