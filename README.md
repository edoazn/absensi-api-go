# absensi-go

Sistem Absensi Mahasiswa berbasis geolocation (rewrite Laravel → Go).
Backend REST API (Gin + GORM + MySQL, JWT) dan panel admin React SPA.

## Stack
- Go 1.25, Gin, GORM (MySQL 8), JWT (access + refresh token rotasi)
- Admin SPA: React 19 + Vite + shadcn (`web/admin`), dilayani di `/admin`
- Migrasi SQL embedded (`internal/database/migrations`)

## Menjalankan (lokal)
```
cp .env.example .env      # sesuaikan DB_* dan JWT_SECRET (>= 32 karakter)
make web-install web-build
make dev                  # atau: go run ./cmd/server
```
- Database dibuat otomatis & migrasi dijalankan saat start.
- `SEED_ON_START=true` untuk data demo (admin: `12345678` / `password`).
- Admin panel: http://localhost:8080/admin/ — API: `/api/v1`, health: `/healthz`.

## Dokumentasi API
Swagger UI: http://localhost:8080/api/documentation/index.html
(regenerasi dengan `make swag`).

## Test
```
make test     # butuh MySQL sesuai .env; DB test dibuat per-proses
make vet
```

## Docker
```
set JWT_SECRET=isi-secret-acak-minimal-32-karakter
docker compose up -d --build
```

## Pengembangan SPA
```
cd web/admin && npm run dev   # proxy /api -> 127.0.0.1:8080
```

Progres & catatan desain: lihat [plan.md](plan.md).
