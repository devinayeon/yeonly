# Yeonly

Yeonly adalah platform Q&A kelas yang anonim dan efemeral (hilang otomatis), dirancang untuk kuliah dan seminar. Dosen dapat membuat ruang dengan kode ruang dan kode QR secara instan; peserta bergabung tanpa perlu membuat akun untuk mengajukan pertanyaan, memberikan upvote, dan mengirimkan umpan balik secara real-time (Paham / Bingung / Terlalu Cepat). Semua data bersifat in-memory dan dihapus saat ruang berakhir, sehingga tidak ada basis data dan tidak ada data yang tersimpan dalam jangka panjang.

## Arsitektur

| Bagian | Teknologi | Lokasi |
| --- | --- | --- |
| Backend | Go, Gorilla WebSocket, API HTTP + WebSocket | `server/` |
| Frontend | Svelte 5, Vite, JavaScript ES modules | `web/` |
| Dokumentasi | Ulasan, spesifikasi, dan rencana kerja | `docs/` |

## Menjalankan secara lokal

**Backend** (melayani API di `:8080`, juga melayani hasil build `web/dist`):

```sh
cd server
go run ./cmd/server
```

**Frontend** (Vite dev server; proxy `/api` dan `/ws` ke `:8080`):

```sh
cd web
npm ci
npm run dev
```

## Validasi

```sh
cd web
npm run validate
```

Perintah ini menjalankan pengecekan tipe yang ketat, regresi Node, dan build produksi.

## Deployment

Rencana lengkap ada di [`docs/ways-of-work/plan/deployment/cloudflare-pages-heroku.md`](docs/ways-of-work/plan/deployment/cloudflare-pages-heroku.md).

- **Frontend** → Cloudflare Pages: root `web`, build `npm ci && npm run build`, output `dist`.
- **Backend** → Heroku: container dari `server/Dockerfile`, satu dyno tanpa autoscaling.
- **Domain**: `nayeonworks.tech` untuk frontend dan `api.nayeonworks.tech` untuk API/WebSocket.

Variabel build frontend (disetel di dashboard Cloudflare Pages):

| Variable | Contoh | Keterangan |
| --- | --- | --- |
| `VITE_API_BASE` | `https://api.nayeonworks.tech` | Kosongkan untuk memakai origin yang sama (dev lokal) |
| `VITE_WS_BASE` | `wss://api.nayeonworks.tech` | Kosongkan untuk memakai origin halaman |

Variabel runtime backend (Heroku):

| Variable | Contoh | Keterangan |
| --- | --- | --- |
| `PORT` | diisi otomatis | Port HTTP |
| `ALLOWED_ORIGINS` | `https://nayeonworks.tech,https://www.nayeonworks.tech` | Kosong berarti semua origin diizinkan (dev) |
| `WEB_DIST` | kosong | Mode API-only; biarkan kosong di Heroku |

Deploy backend:

```sh
cd server
heroku container:push web -a <app>
heroku container:release web -a <app>
```

## Verifikasi kualitas Go (statis)

```sh
cd server
go vet ./...
go test ./...
```

## Lisensi

Belum ditentukan. Silakan tambahkan file `LICENSE` sebelum rilis.
