-- Catatan: down ini gagal bila masih ada baris ber-status 'terlambat' (dev only).
-- Urutan drop/recreate generated column sama seperti up (kompatibel TiDB).
ALTER TABLE attendances
    DROP INDEX attendances_hadir_once_unique;

ALTER TABLE attendances
    DROP COLUMN attended_flag;

ALTER TABLE attendances
    MODIFY status ENUM('hadir','ditolak') NOT NULL;

ALTER TABLE attendances
    ADD COLUMN attended_flag CHAR(1) GENERATED ALWAYS AS (
        IF(status = 'hadir' AND deleted_at IS NULL, 'Y', NULL)
    ) VIRTUAL AFTER method;

ALTER TABLE attendances
    ADD UNIQUE KEY attendances_hadir_once_unique (user_id, schedule_id, attended_flag);

ALTER TABLE attendances
    DROP COLUMN gps_accuracy;
