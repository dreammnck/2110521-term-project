-- One relational database per service (ADR-01). Notifications live in MongoDB (ADR-06).
CREATE DATABASE identity_db;
CREATE DATABASE parcel_db;
CREATE DATABASE announcement_db;
CREATE DATABASE billing_db;
