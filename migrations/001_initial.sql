CREATE TABLE IF NOT EXISTS targets (
  id text PRIMARY KEY, name text NOT NULL, address text NOT NULL,
  healthy boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS route_versions (
  id text NOT NULL, version integer NOT NULL, definition jsonb NOT NULL,
  published_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(id, version)
);
CREATE TABLE IF NOT EXISTS hl7_messages (
  id text PRIMARY KEY, idempotency_key text UNIQUE NOT NULL, message_type text NOT NULL,
  trigger_event text NOT NULL, sending_facility text NOT NULL, raw_digest text NOT NULL,
  raw_object_key text, state text NOT NULL, received_at timestamptz NOT NULL,
  raw_expires_at timestamptz NOT NULL, metadata_expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS deliveries (
  id text PRIMARY KEY, message_id text NOT NULL REFERENCES hl7_messages(id), target_id text NOT NULL,
  status text NOT NULL, attempt integer NOT NULL DEFAULT 0, last_error text,
  ack_code text, ack_digest text, updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS dead_letters (
  id text PRIMARY KEY, delivery_id text NOT NULL REFERENCES deliveries(id), reason text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL, replayed_at timestamptz
);
CREATE TABLE IF NOT EXISTS jobs (
  id text PRIMARY KEY, kind text NOT NULL, reference_id text, status text NOT NULL DEFAULT 'pending',
  attempt integer NOT NULL DEFAULT 0, max_attempts integer NOT NULL, run_after timestamptz NOT NULL,
  locked_at timestamptz, completed_at timestamptz, last_error text, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS deliveries_message_id_idx ON deliveries(message_id);
CREATE INDEX IF NOT EXISTS jobs_claim_idx ON jobs(status, run_after);
