-- Status baru 'terlambat' (deviasi disetujui dari kontrak Laravel):
-- absensi sukses yang masuk pada window (end_time, end_time+5m].
-- attended_flag diperluas agar terlambat juga mengunci slot hadir
-- (tidak bisa submit ulang untuk mengganti status).
--
-- Kompatibel MySQL & TiDB: TiDB menolak MODIFY kolom yang menjadi dependensi
-- generated column, dan tidak mendukung ADD COLUMN ... STORED. Karena itu
-- generated column di-drop dulu, status diubah, lalu attended_flag dibuat
-- ulang sebagai VIRTUAL (unique index di kolom virtual didukung keduanya).
ALTER TABLE attendances
    DROP INDEX attendances_hadir_once_unique;

ALTER TABLE attendances
    DROP COLUMN attended_flag;

ALTER TABLE attendances
    MODIFY status ENUM('hadir','ditolak','terlambat') NOT NULL;

ALTER TABLE attendances
    ADD COLUMN attended_flag CHAR(1) GENERATED ALWAYS AS (
        IF(status IN ('hadir', 'terlambat') AND deleted_at IS NULL, 'Y', NULL)
    ) VIRTUAL AFTER method;

ALTER TABLE attendances
    ADD UNIQUE KEY attendances_hadir_once_unique (user_id, schedule_id, attended_flag);

-- Anti GPS spoof: akurasi laporan device (meter) utk audit & validasi.
ALTER TABLE attendances
    ADD COLUMN gps_accuracy DECIMAL(10, 2) NULL AFTER distance;
