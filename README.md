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

## Verifikasi kualitas Go (statis)

```sh
cd server
go vet ./...
go test ./...
```

## Lisensi

Belum ditentukan. Silakan tambahkan file `LICENSE` sebelum rilis.
