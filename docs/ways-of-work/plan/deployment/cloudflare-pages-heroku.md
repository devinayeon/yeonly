# Rencana Deployment: Cloudflare Pages + Heroku

Status: **draft, belum dieksekusi**
Tanggal: 2026-10-21
Domain utama: `nayeonworks.tech`
Pengganti: konfigurasi Firebase Hosting (`firebase.json`, `.firebaserc`)

## 1. Tujuan

Memindahkan aplikasi Yeonly ke arsitektur dua platform:

- **Frontend statis** (Svelte + Vite) → Cloudflare Pages
- **Backend Go + WebSocket** → Heroku
- **Domain** `nayeonworks.tech` → Cloudflare (apex) dan `api.nayeonworks.tech` → Heroku

Alasan: Firebase Hosting tidak dapat menjalankan server Go, sehingga fitur ruang real-time tidak berfungsi. Heroku mendukung proses long-running dan WebSocket.

## 2. Arsitektur Target

```text
Peserta/Dosen
      │
      ▼
nayeonworks.tech  (Cloudflare Pages, TLS otomatis)
      │  HTML/JS/CSS statis dari web/dist
      │
      │  fetch  https://api.nayeonworks.tech/api/...
      │  WS     wss://api.nayeonworks.tech/ws
      ▼
api.nayeonworks.tech  (Heroku dyno, container Go, TLS otomatis)
      │
      └── state ruang in-memory (efemeral, hilang saat restart)
```

## 3. Prasyarat

- [ ] Akun Cloudflare dengan `nayeonworks.tech` sudah memakai nameserver Cloudflare
- [ ] Akun Heroku dengan billing aktif (tidak ada lagi tier gratis)
- [ ] `git` dan akses push ke `github.com/devinayeon/yeonly`
- [ ] Node 22.12+ (untuk Vite 8) dan Go 1.25 (untuk build lokal)

## 4. Fase 0 — Keputusan dan Pembersihan

| Langkah | Aksi |
| --- | --- |
| 0.1 | Hapus `firebase.json` dan `.firebaserc` dari repo (deployment Firebase dibatalkan) |
| 0.2 | Tambahkan `.firebase/`, `firebase-debug.log*` ke `.gitignore` bila nanti muncul |
| 0.3 | Pastikan `web/dist` tetap di-ignore git; Cloudflare membangunnya sendiri |
| 0.4 | Commit terpisah untuk pembersihan agar riwayat mudah di-*revert* |

## 5. Fase 1 — Perubahan Kode (Blocker)

### 5.1 `web/src/lib/utils/api.js`

Tambahkan basis URL yang dapat dikonfigurasi:

```js
const API_BASE = (import.meta.env?.VITE_API_BASE ?? '').replace(/\/$/, '');

function apiUrl(path) {
  return `${API_BASE}${path}`;
}
```

Ganti ketiga pemanggilan `fetch(path, ...)` menjadi `fetch(apiUrl(path), ...)`.
Tanpa `VITE_API_BASE`, perilaku dev lokal (proxy Vite) tidak berubah.

### 5.2 `web/src/lib/stores/wsStore.js`

```js
const WS_BASE = (import.meta.env?.VITE_WS_BASE ?? '').replace(/\/$/, '');

function wsOrigin() {
  if (WS_BASE) return WS_BASE;
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
}
```

Lalu ganti baris 101–104 agar memakai `wsOrigin()`.
`import.meta.env?.` wajib memakai *optional chaining* karena `node --test` tidak menyediakan `import.meta.env`.

### 5.3 `web/public/_redirects` (baru)

```text
/*  /index.html  200
```

Cloudflare Pages tidak membaca `rewrites` Firebase; file ini menggantikannya agar route SPA tidak 404 saat refresh.

### 5.4 Pengujian

- [ ] `cd web && npm run validate` tetap lulus tanpa env var (fallback same-origin)
- [ ] Tambah kasus uji yang memverifikasi `apiUrl`/`wsOrigin` memakai env bila diset
- [ ] `cd server && go test ./...` tetap lulus

### 5.5 Pengerasan opsional (dapat menyusul)

- Ganti `Access-Control-Allow-Origin: *` dengan allowlist dari env `ALLOWED_ORIGIN`
- Terapkan allowlist serupa pada `CheckOrigin` WebSocket
- Tambah `PING`/`PONG` di server dan client (`wsStore.js`) untuk menjaga koneksi idle

## 6. Fase 2 — Backend di Heroku

### 6.1 Artefak baru di `server/`

- **Dockerfile** (multi-stage, non-root, `CGO_ENABLED=0`, exposure `$PORT`)
- **`.dockerignore`** (`web/`, `*.exe`, `.git`, `.tmp`)
- Health check sudah tersedia di `/health`

Ringkas Dockerfile:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates && adduser -D -u 1001 appuser
USER appuser
COPY --from=builder /server /server
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["/server"]
```

### 6.2 Perintah deploy

```sh
heroku create yeonly-api
heroku container:login
cd server
heroku container:push web -a yeonly-api
heroku container:release web -a yeonly-api
```

`PORT` diisi otomatis oleh Heroku dan sudah dibaca `main.go:20`.

### 6.3 Konfigurasi

| Variable | Nilai | Wajib |
| --- | --- | --- |
| `PORT` | otomatis Heroku | ya |
| `WEB_DIST` | kosong (mode API-only) | tidak |
| `ALLOWED_ORIGIN` | `https://nayeonworks.tech` | nanti |

### 6.4 Batasan platform

- **Satu dyno saja**: state ruang hidup di memori proses; dua dyno = dua dunia terpisah.
- **Restart membuang semua ruang**: Heroku merestart dyno minimal 1×/24 jam dan setiap deploy.
- **Tidak ada persistent disk**: sesuai desain efemeral, tetapi jangan menjanjikan ruang tahan lama.
- **Biaya**: Eco ±$5/dyno/bulan (sleep saat idle); Basic ±$7/bulan (tanpa sleep). Untuk kuliah, Basic lebih aman.

## 7. Fase 3 — Frontend di Cloudflare Pages

### 7.1 Koneksi repo

| Setting | Nilai |
| --- | --- |
| Repository | `devinayeon/yeonly` |
| Production branch | `main` |
| Root directory | `web` |
| Build command | `npm ci && npm run build` |
| Build output | `dist` |
| Node version | `NODE_VERSION=22.12.0` (atau lebih baru) |

### 7.2 Environment variables (build-time, production)

```text
VITE_API_BASE = https://api.nayeonworks.tech
VITE_WS_BASE  = wss://api.nayeonworks.tech
```

Nilai `VITE_*` disematkan saat build, jadi mengubahnya menuntut redeploy. Preview deployment boleh memakai backend staging atau kosong.

### 7.3 Verifikasi preview

- [ ] `https://<hash>.yeonly.pages.dev` memuat landing
- [ ] Refresh di route ruang tidak 404 (uji `_redirects`)
- [ ] Network tab menuju `api.nayeonworks.tech`, bukan origin Pages

## 8. Fase 4 — Domain `nayeonworks.tech`

| Host | Platform | Konfigurasi |
| --- | --- | --- |
| `nayeonworks.tech` | Cloudflare Pages | Pages → Custom domains → tambahkan apex; TLS otomatis karena DNS di Cloudflare |
| `www.nayeonworks.tech` | Cloudflare | Redirect Rule 301 ke apex |
| `api.nayeonworks.tech` | Heroku | `heroku domains:add api.nayeonworks.tech -a yeonly-api`, lalu CNAME ke target `*.herokudns.com` di Cloudflare (DNS only / abu-abu) |

Catatan: record `api` sebaiknya **DNS only** agar Cloudflare tidak mem-proxy WebSocket di atas Heroku; koneksi `wss` tetap TLS dari ACM Heroku.

### Urutan aman

1. Deploy backend, uji `https://yeonly-api.herokuapp.com/health`
2. Tambah domain Heroku, tunggu ACM `OK`
3. Uji `wss://api.nayeonworks.tech/ws` langsung
4. Deploy Pages dengan env var domain final
5. Pasang custom domain apex, verifikasi TLS
6. Aktifkan redirect `www`

## 9. Verifikasi Go-Live

- [ ] `GET /health` → `{"status":"ok","app":"Yeonly"}`
- [ ] Buat ruang sebagai dosen → dapat kode + QR
- [ ] Gabung sebagai peserta dari perangkat lain → WebSocket `connected: true`
- [ ] Kirim pertanyaan → muncul real-time; upvote; pulse Paham/Bingung
- [ ] Refresh halaman ruang → sesi dan peran bertahan
- [ ] Matikan jaringan 5 detik → client reconnect otomatis
- [ ] Deploy ulang → halaman tetap hidup, ruang lama hilang (perilaku yang diterima)

## 10. Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
| --- | --- | --- |
| Restart dyno menghapus ruang | Sesi kelas terputus | Terima sebagai desain efemeral; jadwalkan deploy di luar jam kuliah; pertimbangkan Redis bila perlu |
| Dua dyno / autoscale | Peserta terpisah ke instance berbeda | Kunci di 1 dyno, matikan autoscaling |
| Idle WebSocket diputus router | Indikator terputus berkedip | Auto-reconnect 3 dtk sudah ada; tambah ping server |
| CORS `*` | Origin mana pun dapat memanggil API | Allowlist `ALLOWED_ORIGIN` (Fase 5) |
| Heroku berbayar | Biaya bulanan | Gunakan Basic hanya saat musim kuliah, atau pertimbangkan Cloud Run |
| DNS `api` di-proxy Cloudflare | WebSocket gagal/terputus | Set DNS only (abu-abu) |

## 11. Rollback

| Skenario | Tindakan |
| --- | --- |
| Build Pages rusak | Promote deployment sukses sebelumnya di dashboard Pages |
| Rilis backend rusak | `heroku releases:rollback -a yeonly-api` |
| Perubahan kode bermasalah | `git revert <sha>` lalu push (Pages & Heroku auto-deploy) |
| Domain bermasalah | Lepas custom domain, kembali ke `*.pages.dev` / `*.herokuapp.com` |

## 12. Checklist Eksekusi

- [ ] Fase 0: hapus konfigurasi Firebase, commit
- [ ] Fase 1: env API/WS base, `_redirects`, uji `npm run validate`
- [ ] Fase 2: Dockerfile, `.dockerignore`, deploy Heroku, uji `/health`
- [ ] Fase 3: proyek Pages, env var, uji preview
- [ ] Fase 4: custom domain apex + www + `api`, verifikasi TLS
- [ ] Fase 5: allowlist CORS/WS, heartbeat, pemantauan uptime

## 13. Perkiraan Biaya

| Layanan | Biaya |
| --- | --- |
| Cloudflare Pages | Gratis (bandwidth tak terbatas) |
| Cloudflare DNS + TLS | Gratis |
| Heroku Eco | ±$5/bulan, sleep saat idle |
| Heroku Basic | ±$7/bulan, tanpa sleep |
| Domain | sudah dimiliki |
