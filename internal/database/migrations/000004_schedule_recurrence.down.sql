ALTER TABLE schedules
    DROP COLUMN is_recurring,
    DROP COLUMN repeat_days,
    DROP COLUMN recurrence_end;
