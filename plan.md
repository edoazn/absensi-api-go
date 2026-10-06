# absensi-go — Rencana & Progress Tracking

Rewrite dari Laravel project `absensi-mhs` (Sistem Absensi Mahasiswa Geolocation) ke Golang.
File ini adalah **sumber kebenaran progress** — update setiap kali fitur selesai diimplementasi.

## Keputusan Arsitektur (final)

| Aspek | Keputusan |
|---|---|
| Framework | Gin v1.12 |
| ORM | GORM v1.31 + driver MySQL |
| Auth API | JWT (`golang-jwt/v5`): access token pendek + refresh token di-DB (hash SHA-256, revokeable, rotasi) |
| Admin UI | React SPA satu-satunya (Fase 6); panel HTMX+templ Fase 4 **dihapus total di Fase 8** |
| Migrasi | SQL files embedded (`go:embed`), custom runner + tabel `schema_migrations` |
| DB | MySQL 8 lokal (Laragon), dev: `absensi_go`, test: `absensi_go_test` |
| Excel | excelize (Fase 3) |
| QR | skip2/go-qrcode lokal (Fase 3/4) — tidak ada lagi api.qrserver.com |
| Docs API | swaggo/swag (Fase 5) |
| Admin SPA | React 19 + Vite + shadcn (`web/admin`) konsumsi REST API JWT (Fase 6) |

### Perbaikan desain vs Laravel (user menyetujui redesign)
1. **Anti-duplikat level DB**: kolom generated `attended_flag CHAR(1) AS (IF(status='hadir' AND deleted_at IS NULL,'Y',NULL)) STORED` + unique key `(user_id, schedule_id, attended_flag)` → hanya satu baris `hadir` aktif per user+schedule; retry setelah `ditolak` tetap boleh; soft-deleted tidak menghalangi. Race condition Laravel diperbaiki.
2. **Refresh tokens**: tabel baru; access TTL 15m, refresh TTL 720h (30 hari); logout me-revoke refresh + blacklist jid access sampai exp.
3. **Field hantu dihapus**: `address` (LocationResource) & `credits` (CourseResource) tidak dibawa.
4. **Timezone eksplisit**: semua datetime disimpan sebagai wall-clock `Asia/Jakarta` (DATETIME, bukan TIMESTAMP → bebas masalah 2038), DSN `loc=Asia%2FJakarta`; "today" dihitung via `time.Now().In(loc)`.
5. **Password lama kompatibel**: hash `$2y$` Laravel diverifikasi Go bcrypt setelah replace prefix ke `$2a$`.

### Kontrak bisnis yang harus identik dengan Laravel
- Haversine, earth radius **6371000 m**
- Window absensi ±**5 menit** dari start/end schedule
- 3 metode: `geolocation` / `qr_code` / `attendance_code`
- Status: `hadir` jika distance ≤ radius lokasi, else `ditolak`
- Validasi urutan: terdaftar di kelas → jadwal aktif → belum ada `hadir` → dispatch metode
- Rate limit `POST /attendance`: **10 req/menit**
- Role enum: `admin` / `mahasiswa`; login pakai `identity_number`

## Progress

### Fase 0 — Tracking
- [x] plan.md dibuat, konvensi update disepakati (2026-08-22)

### Fase 1 — Fondasi ✅ SELESAI (2026-08-22)
- [x] Init module `github.com/edoazn/absensi-go` + dependencies core (2026-08-22)
- [x] `.gitignore`, `.env.example`, `.env` lokal (root/root Laragon) (2026-08-22)
- [x] Config loader (`config/config.go`) + import `_ "time/tzdata"` di package config (2026-08-22)
- [x] Migrations SQL embedded: users, classes, class_user, locations, courses, schedules, attendances (+attended_flag generated), refresh_tokens (2026-08-22)
- [x] Models GORM: User, Class (bukan ClassRoom — lihat catatan), Course, Location, Schedule, Attendance, RefreshToken (2026-08-22)
- [x] `internal/database`: EnsureDatabase, Connect, Migrate (koneksi khusus multiStatements=true + USE db) (2026-08-22)
- [x] Seeder idempotent (`internal/seeders`) menerima tz dari config (2026-08-22)
- [x] `cmd/server/main.go`: boot → EnsureDatabase → Migrate → Connect → Seed → `/healthz` + graceful shutdown (2026-08-22)
- [x] Tests lulus 8/8 integrasi MySQL: migrasi idempotent; hadir dobel ditolak DB (1062); retry pasca-ditolak OK; multi-ditolak OK; soft-delete membebaskan slot hadir; user beda sama jadwal OK; toleransi ±5 menit; validity kode absensi (2026-08-22)
- [x] Smoke test server: build + run + seed + GET /healthz = 200 JSON benar (2026-08-22)

### Fase 2 — Auth & Middleware ✅ SELESAI (2026-08-22)
- [x] Token service: JWT access HS256 (claims uid/role/jti) + refresh token random 32B (hash SHA-256 di DB, rotasi transaksional, revoke) (2026-08-22)
- [x] Blacklist jid in-memory untuk access token pasca-logout (lazy purge saat expired) (2026-08-22)
- [x] `services.CheckPassword` kompatibel prefix bcrypt `$2y$` Laravel (replace → `$2a$`) + test (2026-08-22)
- [x] Middleware: JWTAuth (Bearer), RequireAdmin (403), RateLimit token-bucket per-IP in-memory (`golang.org/x/time/rate`) siap dipasang di POST /attendance Fase 3 (2026-08-22)
- [x] Envelope response standar `internal/api`: OK/Created/Fail/FailWithErrors/BindJSON (422 + errors bag terjemahan validator ID) (2026-08-22)
- [x] Router terpusat `internal/server.BuildRouter(Deps)` agar bisa dites httptest; main.go tinggal wiring (2026-08-22)
- [x] Endpoint: POST /api/v1/auth/login, /auth/refresh, /auth/logout, GET /api/v1/me, GET /healthz (ping DB), GET /api/v1 (info) (2026-08-22)
- [x] Test lulus: 9 test handler auth (login sukses/salah/unknown/422, me 200/401/tampered, rotasi refresh + reuse ditolak, logout blacklist access & revoke refresh), 5 test middleware (admin guard, missing/tampered bearer, blacklist, rate limit burst + isolasi IP), 2 test password — semua terhadap MySQL asli via httptest (2026-08-22)
- [x] Smoke test server asli 2x run: seeder idempotent, login admin+mahasiswa → /me → logout OK; refresh invalid → 401 (2026-08-22)

### Fase 3 — Domain Absensi ✅ SELESAI (2026-08-22)
- [x] GeolocationService: Haversine earth radius 6371000m + IsWithinRadius; 6 test properti lulus (identitas=0, simetri, determinisme, 1° lat ≈111km, segitiga, boundary radius inklusif) (2026-08-22)
- [x] AttendanceService.Process: urutan validasi jadwal ada → terdaftar di kelas → window ±5 menit aktif → dispatch metode (2026-08-22)
- [x] Persist race-safe: transaction + `SELECT id FROM schedules FOR UPDATE` + cek hadir existing; backstop unique constraint `attended_flag` (1062 dipetakan ke ErrAlreadyAttended) (2026-08-22)
- [x] 3 metode: geolocation (hadir/ditolak, distance dibulatkan 2dp), qr_code (match UUID), attendance_code (match case-insensitive + expiry) — pesan error identik dengan versi Laravel (2026-08-22)
- [x] Endpoint mahasiswa: POST /attendance (rate limit 10/menit burst), GET /attendance/history?page&per_page (meta pagination, desc), GET /schedules/today (range harian tz Jakarta, asc, is_active flag) (2026-08-22)
- [x] Admin API: GET/POST/PUT /locations; GET /schedules (+filter class_id/course_id); POST /schedules; POST /schedules/:id/generate-code (default 30m, clamp 1–1440, kode 6 char uppercase crypto/rand); POST /schedules/:id/generate-qr (UUID rotate) (2026-08-22)
- [x] Reports admin: GET /reports/attendance?start_date&end_date&schedule_id (format YYYY-MM-DD, range created_at [start, end+24h)); GET /reports/attendance/export via excelize (13 kolom Indonesia, sheet "Absensi", filename timestamp) (2026-08-22)
- [x] Tests baru: 11 integrasi attendance + 7 admin/CRUD/report — termasuk verifikasi xlsx asli via excelize.OpenReader (B1="Nama Mahasiswa", H2 status) dan rate limit request ke-11 = 429 (2026-08-22)
- [x] **Fix interferensi test paralel antar-paket** yang memakai satu DB fisik: database test kini per-proses (`absensi_go_test_<pid>`) — suite stabil 5/5 paket ok pada -count=1 dua kali beruntun (2026-08-22)

### Fase 4 — Admin Panel (HTMX + templ) ✅ SELESAI (2026-08-22)
- [x] templ v0.3.1020 terinstall (`go install .../cmd/templ`), runtime `a-h/templ` di go.mod; generate via `templ generate ./internal/views/` (2026-08-22)
- [x] Migrasi `000002_admin_sessions` + model AdminSession + service sesi (token random 32B, hash SHA-256, TTL 12 jam, prune expired) (2026-08-22)
- [x] Session cookie httpOnly SameSite=Lax terpisah dari JWT API; middleware RequireSession; logout destroy sesi (2026-08-22)
- [x] Views templ: layout sidebar + login page + dashboard (4 kartu statistik, doughnut Chart.js lokal data-attrs, recent 5 absensi, jadwal hari ini berbadge Berlangsung/Akan Datang/Selesai) (2026-08-22)
- [x] CRUD UI Users (form inline + multi-select kelas + validasi duplikat NIM/email + password opsional saat edit) (2026-08-22)
- [x] CRUD UI Kelas (multi-pilih mahasiswa), Mata Kuliah (kode unik uppercase), Lokasi (validasi range lat/lon/radius) (2026-08-22)
- [x] Jadwal: list + filter kelas/MK, create/edit datetime-local, delete soft; **Generate Kode** via HTMX fragment tanpa reload; **QR PNG lokal** `/admin/schedules/:id/qr.png` (skip2/go-qrcode, no-store) + rotate token (2026-08-22)
- [x] Absensi read-only: filter status/metode/range tanggal + pagination; Export Excel reuse ReportHandler.ExportExcel di bawah guard sesi (2026-08-22)
- [x] Aset statis lokal: htmx.min.js + chart.umd.js (web/static); Tailwind v4 browser CDN sementara (catatan: swap ke CLI build saat production — lihat Catatan) (2026-08-22)
- [x] Tests: 9 test UI baru (login flow penuh incl. mahasiswa ditolak & logout mematikan sesi, guard semua halaman, users CRUD + pivot, duplikat NIM error ramah, assignment siswa kelas, fragmen HTMX kode persist DB, QR PNG magic bytes + rotasi, filter badge absensi) — semua lulus (2026-08-22)
- [x] Smoke test server asli: login 200→302→dashboard render, tombol kode muncul, QR PNG 200 image/png, logout 302 (2026-08-22)
- Suite penuh stabil: 5/5 paket ok (-count=1)

### Fase 5 — Docs, Testing, Deploy ✅ SELESAI (2026-10-06)
- [x] Swagger annotations semua handler + serve /api/documentation
- [x] Makefile lengkap (dev/build/test/migrate/seed/swag)
- [x] README + Dockerfile + docker-compose (opsional)

### Fase 6 — Admin SPA (React + shadcn) ✅ SELESAI (2026-08-23)
**Backend REST baru (paritas dengan HTMX panel):**
- [x] Users CRUD: GET (+filter role) / POST / PUT / DELETE `/api/v1/users`; pivot class_ids hanya utk mahasiswa; validasi duplikat NIM/email + rule password (min 6, opsional saat edit) identik HTMX (2026-08-23)
- [x] Classes CRUD: GET (preload Students) / POST / PUT / DELETE `/api/v1/classes` dengan sync multi-siswa transaksional (2026-08-23)
- [x] Courses CRUD: GET / POST / PUT / DELETE `/api/v1/courses`; kode uppercase unik (exclude self saat update); location_room nullable (2026-08-23)
- [x] Schedules: tambah PUT `/schedules/:id` (validasi FK + end>start, format waktu fleksibel) dan DELETE `/schedules/:id` (soft delete) (2026-08-23)
- [x] GET `/api/v1/admin/dashboard`: totals (users/classes/courses/locations/today/hadir/ditolak) + recent 5 absensi + jadwal hari ini berbadge is_active (2026-08-23)
- [x] GET `/schedules/:id/qr.png` versi JWT-guarded (Bearer, bukan cookie sesi) untuk konsumsi SPA (2026-08-23)
- [x] Index jadwal kini menyertakan class_id/course_id/location_id mentah agar form edit SPA bisa memetakan select (2026-08-23)
- [x] Tests `masterdata_test.go`: 6 test integrasi (user CRUD roundtrip+filter, validasi/duplikat user, class CRUD pivot sync, course CRUD uppercase+duplikat, dashboard stats, QR PNG magic bytes via Bearer) (2026-08-23)
- [x] Fix flake lama `TestReportFilteringAndExport`: todayRow pakai `dayStart.Add(12h)` bukan `now-1h` (sebelumnya gagal saat dijalankan lewat tengah malam) (2026-08-23)
- Suite penuh 5/5 paket ok (-count=1)

**Frontend (`web/admin`):**
- [x] Komponen shadcn diinstall via MCP registry @shadcn → CLI: button, input, label, card, table, dialog, select, badge, dropdown-menu, alert-dialog, sonner, skeleton, separator, sidebar (+sheet/tooltip dependensi), spinner, checkbox, textarea, empty, avatar — style radix-nova (2026-08-23)
- [x] `lib/api.ts`: wrapper fetch envelope {success,data,message,errors}; access+refresh token di localStorage; auto-refresh sekali saat 401 lalu retry; apiBlob untuk QR PNG & export xlsx; login/logout/fetchMe (2026-08-23)
- [x] `lib/auth.tsx`: AuthProvider; login menolak non-admin (token langsung dibuang); logout memanggil API revoke (2026-08-23)
- [x] Shell: AppLayout = SidebarProvider + AppSidebar (7 nav NavLink active state) + AppHeader (SidebarTrigger + avatar dropdown logout); RequireAdmin guard redirect /login (2026-08-23)
- [x] Halaman: Login (card center, error inline), Dashboard (4 kartu stat + hadir/ditolak + recent table + jadwal hari ini), Pengguna (dialog form + checkbox kelas muncul saat role=mahasiswa), Kelas (checkbox multi-mahasiswa), Mata Kuliah (kode auto-uppercase), Lokasi, Jadwal (filter kelas/MK, create/edit datetime-local, dialog Generate Kode dgn minutes_valid + tampil kode besar, dialog QR dari blob + Rotate Token), Absensi (filter tanggal/jadwal + unduh Excel via blob) (2026-08-23)
- [x] Vite proxy `/api` → 127.0.0.1:8080 (dev); build outDir `web/static/admin` (deploy nanti); tsconfig.app.json hapus `baseUrl` deprecated TS6 (cukup paths relatif) (2026-08-23)
- [x] Verifikasi: `tsc -b` bersih; oxlint hanya warning pola bawaan shadcn/fetch-on-mount; `npm run build` sukses; smoke test end-to-end via proxy Vite: login seeder 12345678/password → /me → dashboard → users/classes → courses create/delete → report → export xlsx 200 OK; suite Go tetap hijau (2026-08-23)
- [x] **Fix white screen saat dilayani dari Go server**: build produksi kini pakai `base: '/static/admin/'` (dev tetap `/`), BrowserRouter `basename={import.meta.env.BASE_URL}`, dan router.go mengganti `router.Static` dengan handler custom — berkas ada → kirim; path `/static/admin/*` tak dikenal (deep-link SPA) → fallback index.html; `GET /` redirect ke `/static/admin/`. Aset lama HTMX (`/static/js/...`) tetap jalan (2026-08-23)
- [x] **Fix white screen kedua**: `Tooltip must be used within TooltipProvider` — sidebar shadcn (radix-nova) pakai Tooltip di SidebarMenuButton sehingga seluruh halaman ber-sidebar crash saat render; solusi bungkus `<Routes>` dengan `TooltipProvider` di App.tsx (halaman login lolos karena tanpa sidebar → white screen hanya terasa setelah login). Verifikasi E2E headless Chrome: seed localStorage token via halaman sementara → dump DOM dashboard = sidebar + 9 card + recent + jadwal hari ini ter-render (2026-08-23)

### Fase 7 — Hardening Backend ✅ SELESAI (2026-08-23)
> Keputusan desain baru yang MENYIMPANG dari kontrak Laravel (user menyetujui):
- [x] **Rate limit POST /auth/login**: 10 burst / 10-per-menit per-IP (token bucket sama dengan attendance) — menutup brute-force NIM/password; test 11th request = 429 (2026-08-23)
- [x] **QR wajib koordinat** (anti share-screenshot): metode qr_code kini WAJIB menyertakan latitude/longitude (binding required_if + backstop di service); jarak dihitung & divalidasi terhadap radius lokasi → dalam radius = hadir (dgn distance), luar radius = ditolak (row tersimpan utk audit). ⚠️ BREAKING untuk client lama: QR tanpa koordinat → 422 "Koordinat lokasi wajib disertakan..." (2026-08-23)
- [x] **Pagination GET /reports/attendance**: page/per_page (default 15, clamp 1..100), response `{items, meta:{current_page,per_page,total,last_page}}` konsisten dgn history mahasiswa; Export Excel tetap full-range. SPA Absensi diupdate: info halaman + tombol Sebelumnya/Berikutnya (2026-08-23)
- [x] Tests baru: TestQrCodeFlow (revisi — QR sukses kini bawa distance), TestQrCodeRequiresCoordinates, TestQrCodeOutsideRadiusRejected (verifikasi row ditolak + lat/distance tersimpan), TestLoginRateLimited, TestReportPagination (meta per halaman) — suite 5/5 paket ok -count=1 (2026-08-23)
- Backlog lanjutan dari review (belum dikerjakan): rate-limit per-user utk route auth'd, deteksi GPS spoof (accuracy/geo-velocity), guard hapus diri-sendiri/admin-terakhir, CORS utk mobile dev, Redis blacklist multi-instance, jadwal recurring, status terlambat, audit log admin

### Fase 10 — GET /schedules/:id + Test PUT/DELETE Schedule ✅ SELESAI (2026-08-24)
> Item backlog Fase 7. `PUT`/`DELETE` /schedules/:id sebenarnya sudah ada sejak Fase 6 tapi tanpa test sama sekali.
- [x] Handler baru `ScheduleHandler.Show` (schedule.go): preload ClassRoom/Course/Location Unscoped + payload identik dengan item Index via helper bersama `scheduleItem()` (Index direfaktor memakainya); 400 ID non-numerik, 404 tak ada/soft-deleted (2026-08-24)
- [x] Route: `admin.GET("/schedules/:id", Show)` — coexist aman dengan `/schedules/today` statis milik mahasiswa (gin prioritas statis) (2026-08-24)
- [x] **Bug ketemu test**: `Destroy` lama memakai `db.Delete()` mentah — DELETE ke id yang sudah terhapus/nonexistent tetap 200 karena RowsAffected tak dicek; fix dengan `First()` dulu → 404 (2026-08-24)
- [x] Tests masterdata_test.go: TestScheduleShowEndpoint (payload lengkap + is_active, 404 unknown, 400 abc), TestScheduleUpdateRoundTripAndValidation (update persist diverifikasi via Show, FK kelas invalid 422, end<=start 422 errors bag, format waktu invalid 422), TestScheduleDeleteSoftDeletesRow (row lunak tersisa via Unscoped, scoped=0, Show pasca-delete 404, delete kedua 404) — semua lulus (2026-08-24)
- [x] Verifikasi: gofmt/vet bersih, suite 7 paket ok (-count=1); smoke server asli: GET detail 200 payload penuh, 999999→404, abc→400 (2026-08-24)

### Fase 8 — Removal Admin Panel HTMX ✅ SELESAI (2026-08-23)
> Keputusan: React SPA (Fase 6) kini satu-satunya admin UI; seluruh jejak HTMX dihapus bersih.
- [x] Hapus 6 file `internal/handlers/ui_*.go` (+ `ui_test.go` 9 test UI) dan seluruh `internal/views/` (7 .templ + generated + data.go) — ±1.500 baris (2026-08-23)
- [x] Hapus model+service `AdminSession`, migrasi `000002_admin_sessions.{up,down}.sql`; dev DB dibersihkan manual (`DROP TABLE admin_sessions` + hapus baris schema_migrations v2 via mysql CLI Laragon) (2026-08-23)
- [x] Router: buang grup `uiGuarded` + `/admin/login` versi sesi; tambah `redirectAdminLegacy()` (`router.Any("/admin")` & `"/admin/*rest"`) → 302 ke SPA `/static/admin/...` (section root utk users/classes/courses/locations/schedules/attendances, login → login, sisanya dashboard; query lama dibuang; POST dari tab basi pun dialihkan) (2026-08-23)
- [x] Hapus aset `web/static/js/{htmx.min.js,chart.umd.js}`; `go mod tidy` → `a-h/templ` keluar; `skip2/go-qrcode` TETAP (dipakai QR PNG API JWT) (2026-08-23)
- [x] Ekstrak helper `paramUint` (sebelumnya nyempil di ui_users.go, dipakai class/course/user handler API) → `internal/handlers/params.go` (2026-08-23)
- [x] Verifikasi: build/vet/gofmt bersih; suite 5 paket ok (-count=1); smoke server asli: semua varian /admin/* 302 tepat sasaran, htmx.min.js 404, deep-link SPA 200, login API admin OK. GOTCHA: server lama dari sesi sebelumnya masih memegang port 8080 sehingga binary baru gagal bind — cek `netstat -ano | grep :8080` dulu saat hasil curl "aneh" (2026-08-23)

### Fase 9 — SPA di `/admin` (hapus warisan prefix `/static`) ✅ SELESAI (2026-08-24)
> Keputusan (revisi 2x dalam satu hari, user yang memutuskan): prefix `/static` warisan HTMX dihapus; SPA dilayani di **`/admin`** — bukan root `/` (sempat dieksekusi lalu dibatalkan karena URL panel harus tetap ber-prefix `/admin`). Bonus: URL panel HTMX lama (`/admin/users`, dst) otomatis menjadi rute asli React TANPA redirect.
- [x] Router: hapus route `/static/*filepath` + `serveStatic()`; SPA dilayani `router.GET("/admin", spa)` + `/admin/*filepath` → `spaHandler("./web/dist")`: file ada (anti-traversal via Abs+HasPrefix/EqualFold terhadap rootAbs) → serve (`/assets/*` ber-hash `immutable` 1 tahun, lainnya `no-cache`), path ber-ekstensi tak ada → 404 polos, deep-link/dir → index.html. `GET /` → 302 `/admin/`; path di luar `/admin` → 404 polos via NoRoute, kecuali `/api*` → 404 JSON envelope (kontrak REST utuh). Redirect tersisa: `/static/admin*` → 302 ke `/admin*` (bookmark era dev) (2026-08-24)
- [x] Vite: `base: '/admin/'`, outDir `web/static/admin` → `web/dist`; App.tsx `BrowserRouter basename={import.meta.env.BASE_URL}` (= `/admin/`); `web/static/` dihapus; `.gitignore` + `web/dist/` (build output tak di-commit — deploy wajib `npm run build` dulu) (2026-08-24)
- [x] Test baru `internal/server/spa_test.go`: fixture `t.TempDir()/web/dist` + chdir per-test (tak butuh MySQL/npm build); subtest fallback (/admin root, deep-link, aset immutable, no-cache, 404 ekstensi, path luar /admin 404, API 404 JSON, traversal ditolak, / → 302) + 6 kasus redirect `/static/admin*` — semua lulus (2026-08-24)
- [x] Verifikasi: tsc/vet/gofmt bersih; suite 6 paket ok (-count=1); smoke server asli: `/`→302 `/admin/`, `/admin` & `/admin/users` 200 HTML, aset immutable, `/admin/missing.css` 404, `/users` 404, `/api/v1/xxx` 404 JSON, `/static/admin/users`→302 `/admin/users`; E2E headless Chrome (seed page HARUS di bawah `/admin/` sekarang): dump DOM `/admin/users` = tabel 3 user seeder + tombol Tambah Pengguna (2026-08-24)
- Catatan: percobaan pertama "SPA di root `/`" sempat selesai penuh (test+smoke+E2E hijau) lalu di-revisi ke `/admin` atas masukan user di hari yang sama — tidak ada jejak root-mount yang tersisa.

## Catatan Implementasi (jebakan & solusi)

- Windows + Go: `time.LoadLocation("Asia/Jakarta")` butuh tzdata → import blank `_ "time/tzdata"` di package `config` (bukan main) agar semua binary/test dapat jaminan.
- Kredensial MySQL dev Laragon: root/root@tcp(127.0.0.1:3306). Catatan: klien CLI `mysql.exe` gagal via host-match `localhost`, tapi TCP 127.0.0.1 dari Go/app normal.
- **Migrasi butuh koneksi terpisah**: go-sql-driver menolak multi-statement secara default → `Migrate(cfg)` membuka koneksi sendiri dengan `multiStatements=true` + eksekusi `USE db`. GORM Exec tidak dipakai untuk DDL file penuh.
- MySQL DDL tidak transaksional: jika migrasi gagal di tengah, versi TIDAK dicatat → drop database lalu migrasi ulang (dev only).
- Model dinamai `Class` (bukan ClassRoom) karena GORM menurunkan kolom join many2many dari nama model; tabel pivot `class_user` pakai `class_id` — rename tipe memperbaiki derivasi tanpa tag eksplisit. TableName() = "classes".
- Relasi GORM eksplisit `foreignKey:UserID/ClassID` pada has-many untuk hindari error parsing relasi sirkular.
- GORM + kolom generated `attended_flag`: TIDAK didefinisikan di struct Attendance (dikelola penuh oleh DB).
- ENUM MySQL dipertahankan untuk status/method (paritas dengan skema Laravel).
- Seeder: idempotent via Where+FirstOrCreate natural key (identity_number / course_code / name); jadwal hari ini dicek per class+course+tanggal; QR allday DEMO01 valid 7 hari.
- **GOTCHA GORM FirstOrCreate**: `FirstOrCreate(&dest, attrs)` menggabungkan attrs ke kondisi SELECT juga! Karena attrs berisi hash bcrypt baru (selalu beda), lookup tidak pernah match → INSERT ulang → 1062 saat server restart. Solusi: helper `ensureXxx` manual — `Where(natural_key).First()` → jika ErrRecordNotFound lalu `Create` dengan data penuh. JANGAN pernah taruh field volatile di kondisi.
- **Test DB per-proses**: paket Go test berjalan paralel antar-binari; satu DB bersama menyebabkan TRUNCATE saling menginjak (fail acak). Solusi: `absensi_go_test_<pid>` via `os.Getpid()`. Sisa DB lama dibersihkan manual:
  `SELECT ... FROM information_schema.schemata WHERE schema_name LIKE 'absensi_go_test_%'` → DROP.
- **xlsx assertion**: isi file Excel terkompresi ZIP — jangan cari teks mentah di body; buka dengan `excelize.OpenReader` lalu GetCellValue.
- Handler yang menerima body opsional (generate-code API): JANGAN pakai ShouldBindJSON biasa (body kosong = EOF error). Pakai `c.GetRawData()` → trim → jika ada, json.Unmarshal + validasi manual clamp.
- **gin route static vs param**: `/users/new` dan `/users/:id/edit` bisa coexist (gin modern prioritas statis).
- **Form binding gin**: butuh tag `form:"..."` (tag json tidak dipakai untuk urlencoded). Body opsional tetap pakai pola GetRawData.
- Logout Laravel menghapus SEMUA token user; versi Go: revoke refresh milik sesi itu saja + blacklist jid access yang bersangkutan (in-memory; multi-instance nanti perlu Redis).
- Perintah dev: `go run ./cmd/server`, `go test ./...`, `go vet ./...`, `gofmt -l .`.
- **shadcn MCP + CLI**: registry `@shadcn` bawaan CLI — JANGAN tulis ke `registries` di components.json (validasi gagal "Invalid configuration"); cukup panggil `npx shadcn add <nama>` tanpa prefix. MCP search/list berguna untuk eksplorasi item & blocks.
- **Vite v8 di Windows** listen IPv6 `[::1]` saja secara default → curl smoke test harus pakai `localhost`, bukan `127.0.0.1`.
- **TypeScript 6**: `baseUrl` deprecated (error TS5101) → hapus, `paths` relatif `"@/*": ["./src/*"]` sudah cukup.
- SPA auth: refresh token di localStorage + auto-rotate via `/auth/refresh` saat 401 (retry sekali); logout mem-blacklist jid access + revoke refresh milik sesi. QR PNG & export xlsx di-fetch sebagai blob dengan header Bearer (bukan link langsung).
- ~~**SPA served dari Gin**: Vite `base` absolut (`/static/admin/`)...~~ **USANG — disuperseed Fase 9**: SPA kini di `/admin` via route eksplisit; Vite `base: '/admin/'`, `basename` React Router dari `import.meta.env.BASE_URL` — keduanya WAJIB sinkron dengan prefix di router.go.
- **SPA fallback via gin NoRoute (Fase 9)**: catch-all `/*filepath` PANIC konflik dgn rute statis lain di radix tree gin — untuk mount di root satu-satunya cara adalah NoRoute. Versi akhir memilih mount `/admin/*filepath` (route eksplisit, tanpa NoRoute utk SPA) sehingga masalah ini menghilang; NoRoute kini hanya menjaga 404 JSON utk `/api*`.
- **Jebakan anti-traversal prefix check**: request ke root mount me-resolve tepat ke root dir (tanpa trailing `\`) sehingga `HasPrefix(full, rootAbs+sep)` = false → ikut ke-404. Solusi: hitung juga kesetaraan (`EqualFold` utk Windows case-insensitive) selain prefix.
- **http.ServeFile kanonialisasi**: `GET /…/index.html` otomatis 301 → `./`; itu perilaku bawaan net/http, bukan bug router.
- **Test handler berbasis cwd-relative root**: pola fixture `t.TempDir()/web/dist` + helper `chdir(t)` per-test membuat test SPA tak bergantung hasil npm build maupun MySQL; jangan panggil `t.Parallel()` di paket yang memakai chdir.
- Halaman utilitas sementara (mis. seeding localStorage untuk E2E) harus ditaruh DI DALAM dist yang dilayani — sekarang berarti URL-nya `/admin/<nama>.html`; path luar `/admin` sudah tidak dilayani (404).
- **shadcn radix-nova + sidebar**: komponen Sidebar memakai Tooltip internal → app WAJIB dibungkus `TooltipProvider` (sesuai snippet instalasi shadcn), kalau tidak seluruh halaman ber-sidebar crash `Tooltip must be used within TooltipProvider` — dan karena LoginPage tidak pakai sidebar, bug ini hanya kelihatan SETELAH login (putih tanpa jejak di halaman login).
- Debug white screen SPA: uji dengan headless Chrome `--dump-dom --virtual-time-budget` untuk memisahkan masalah server vs ekstensi browser; atribut aneh seperti `bis_register`/script `extension://` di DOM = injeksi ekstensi, bukan bug aplikasi.
- curl `-I` = HEAD request — route gin yang hanya register GET akan 404 untuk HEAD walai GET normal; jangan panik, tes ulang dengan GET.
- Test laporan: jangan buat baris "hari ini" dengan `now.Add(-time.Hour)` — melewati tengah malam membuatnya jatuh ke kemarin; pakai offset dari dayStart (`dayStart.Add(12*time.Hour)`).

## Referensi Project Lama

- Path: `D:\applications\Laravel projects\absensi-mhs`
- Kontrak Flutter client: `guide.md` di repo lama
- 45 property tests Pest = acuan perilaku (tests/Property/)
