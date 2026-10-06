-- Jadwal mingguan berulang. start_time/end_time tetap menyimpan OCCURRENCE
-- PERTAMA; tanggal efektif per-minggu diturunkan dari repeat_days (CSV hari
-- 0=Minggu .. 6=Sabtu). recurrence_end opsional membatasi seri.
ALTER TABLE schedules
    ADD COLUMN is_recurring TINYINT(1) NOT NULL DEFAULT 0 AFTER end_time,
    ADD COLUMN repeat_days VARCHAR(13) NULL AFTER is_recurring,
    ADD COLUMN recurrence_end DATE NULL AFTER repeat_days;

CREATE INDEX idx_schedules_recurring ON schedules (is_recurring, start_time);
