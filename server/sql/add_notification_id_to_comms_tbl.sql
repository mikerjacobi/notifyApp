
ALTER TABLE communications ADD COLUMN notification_id varchar(36) DEFAULT "" AFTER comms_id;
CREATE INDEX notification_id_index ON communications (notification_id);
