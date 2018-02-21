DELETE FROM services WHERE deleted = TRUE;

ALTER TABLE services
  DROP COLUMN deleted,
  DROP COLUMN delete_time;