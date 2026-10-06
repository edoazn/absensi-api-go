# Panduan Deploy API Absensi 100% Gratis

Dokumen ini adalah panduan langkah demi langkah untuk mengonline-kan backend (Golang) dan frontend (React) `absensi-go` secara gratis agar bisa diakses oleh aplikasi mobile.

Kombinasi yang kita gunakan:
- **TiDB Serverless** (Untuk Database MySQL - 5GB Gratis Selamanya)
- **Koyeb** (Untuk hosting Server Go + Docker - 1 Web Service Gratis Selamanya, Tidak bisa tidur/sleep)

---

## 1. Persiapkan Akun GitHub
Pastikan *source code* `absensi-go` ini sudah Anda upload (push) ke repository GitHub Anda (bisa public maupun private).

---

## 2. Setup Database di TiDB Serverless
TiDB adalah database modern yang kompatibel 100% dengan MySQL.
1. Buka [TiDB Cloud](https://tidbcloud.com) dan buat akun (bisa *Sign Up with GitHub*).
2. Buat cluster baru, pilih tipe **Serverless**.
3. Pilih region terdekat dengan pengguna (misalnya Singapura/Singapore/ap-southeast-1) agar API berjalan cepat.
4. Klik **Create**.
5. Setelah cluster dibuat, klik tombol **Connect** di pojok kanan atas.
6. Anda akan diminta membuat password untuk user database Anda. **Simpan password tersebut baik-baik!**
7. Di layar koneksi, ubah opsi "Connect With" menjadi **General** atau **Go**.
8. Catat kredensial berikut:
   - **Host:** (misal: `gateway01.ap-southeast-1.prod.aws.tidbcloud.com`)
   - **Port:** `4000`
   - **User:** (misal: `2k3j4h234.root`)
   - **Password:** (Password yang Anda buat tadi)

---

## 3. Setup Server di Koyeb
Koyeb akan membaca kode dari GitHub Anda, menjalankan Dockerfile kita, dan menyalakannya.

1. Buka [Koyeb.com](https://app.koyeb.com/) dan buat akun menggunakan GitHub.
2. Di dashboard, klik tombol **Create Service** (Create Web Service).
3. Pilih metode deployment: **GitHub**.
4. Pilih repository `absensi-go` Anda.
5. Pada bagian *Builder*, pilih **Dockerfile** (Koyeb akan otomatis menemukan `Dockerfile` kita di root project).
6. Di bagian *Instance*, pilih **Free / Eco** (pastikan tagihannya $0/month).
7. Buka bagian **Environment Variables** (Ini yang paling penting). Masukkan variabel berikut:

| Key | Value |
| --- | --- |
| `APP_ENV` | `production` |
| `PORT` | `8080` *(Koyeb akan mendengarkan port ini)* |
| `DB_HOST` | *(Host TiDB yang Anda catat, misal: gateway01...tidbcloud.com)* |
| `DB_PORT` | `4000` |
| `DB_USERNAME` | *(User TiDB yang Anda catat)* |
| `DB_PASSWORD` | *(Password TiDB Anda)* |
| `DB_DATABASE` | `test` *(Database bawaan TiDB biasanya bernama test)* |
| `DB_ARGS` | `tls=true` *(Wajib diisi agar koneksi TiDB aman/diterima)* |
| `JWT_SECRET` | *(Ketik teks acak minimal 32 karakter, misal: `sangat-rahasia-jangan-kasih-tau-siapapun`)* |
| `SEED_ON_START`| `true` *(Wajib di-set "true" agar otomatis mengisi data seeder akun admin saat pertama kali nyala)* |
| `CORS_ORIGINS` | `*` *(Mengizinkan semua aplikasi mobile untuk memanggil API ini)* |

8. Di bagian *Port*, pastikan diset ke **8080** (Sesuai dengan default port Go API kita).
9. Klik **Deploy**.

---

## 4. Testing dan Selesai!
Koyeb akan mulai men-*download* kode Anda, mem-*build* image Docker (bisa memakan waktu 2-3 menit), lalu menjalankannya.
Setelah statusnya hijau (**Healthy**), Koyeb akan memberikan URL publik untuk aplikasi Anda (contoh: `https://absensi-go-yourname.koyeb.app`).

**Cara ngetesnya:**
1. Buka URL tersebut di browser: `https://absensi-go-yourname.koyeb.app/admin/`
2. Anda harusnya melihat panel login admin React SPA!
3. Login menggunakan:
   - NIM: `12345678`
   - Password: `password`
4. Untuk dokumentasi Swagger/API Mobile, buka: `https://absensi-go-yourname.koyeb.app/api/documentation/index.html`

Kini API backend (Golang) dan frontend Admin SPA sudah online gratis 24 jam! Tim mobile dev sudah bisa mulai tembak endpoint API tersebut dari Flutter/Kotlin/React Native.
