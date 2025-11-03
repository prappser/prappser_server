-- Add application_id column back for better query performance
-- (previously removed in 000006, but needed for efficient ordering)
ALTER TABLE events ADD COLUMN application_id TEXT;

-- Populate application_id from JSON data for existing rows
UPDATE events SET application_id = json_extract(data, '$.applicationId');

-- Add sequence_number column for strict event ordering per application
ALTER TABLE events ADD COLUMN sequence_number INTEGER NOT NULL DEFAULT 0;

-- Create index for efficient querying by application and sequence
CREATE INDEX idx_events_app_sequence ON events(application_id, sequence_number);
