-- Drop index dulu & satu kolom per ALTER agar kompatibel TiDB.
DROP INDEX idx_schedules_recurring ON schedules;

ALTER TABLE schedules DROP COLUMN recurrence_end;
ALTER TABLE schedules DROP COLUMN repeat_days;
ALTER TABLE schedules DROP COLUMN is_recurring;
