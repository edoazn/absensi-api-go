-- Status baru 'terlambat' (deviasi disetujui dari kontrak Laravel):
-- absensi sukses yang masuk pada window (end_time, end_time+5m].
-- attended_flag diperluas agar terlambat juga mengunci slot hadir
-- (tidak bisa submit ulang untuk mengganti status).
ALTER TABLE attendances
    MODIFY status ENUM('hadir','ditolak','terlambat') NOT NULL;

ALTER TABLE attendances
    MODIFY attended_flag CHAR(1) GENERATED ALWAYS AS (
        IF(status IN ('hadir', 'terlambat') AND deleted_at IS NULL, 'Y', NULL)
    ) STORED;

-- Anti GPS spoof: akurasi laporan device (meter) utk audit & validasi.
ALTER TABLE attendances
    ADD COLUMN gps_accuracy DECIMAL(10, 2) NULL AFTER distance;
