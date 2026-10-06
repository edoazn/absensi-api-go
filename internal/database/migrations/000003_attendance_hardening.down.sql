-- Catatan: down ini gagal bila masih ada baris ber-status 'terlambat' (dev only).
ALTER TABLE attendances
    MODIFY status ENUM('hadir','ditolak') NOT NULL,
    MODIFY attended_flag CHAR(1) GENERATED ALWAYS AS (
        IF(status = 'hadir' AND deleted_at IS NULL, 'Y', NULL)
    ) STORED;

ALTER TABLE attendances
    DROP COLUMN gps_accuracy;
