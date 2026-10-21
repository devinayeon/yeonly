# Rencana Deployment: Cloudflare Pages + Heroku

Status: **Fase 0–2 dan 5 diterapkan di repo; jalur deploy Heroku tanpa Docker disiapkan; Fase 3–4 menunggu kredensial Cloudflare/Heroku**
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

### 5.5 Pengerasan (sudah diterapkan)

- `Access-Control-Allow-Origin` kini mengikuti allowlist env `ALLOWED_ORIGINS`; kosong berarti `*` seperti sebelumnya
- `CheckOrigin` WebSocket memakai allowlist yang sama
- Heartbeat sudah tersedia sejak awal: server mengirim ping tiap 54 detik (`server/internal/ws/client.go`)

## 6. Fase 2 — Backend di Heroku

### 6.1 Artefak container

- `server/Dockerfile` — image API-only untuk build lokal dan `heroku container:push`
- `server/.dockerignore` — konteks build untuk `server/Dockerfile`
- `Dockerfile.heroku` — image API-only yang dibangun Heroku dari root repo
- `heroku.yml` — memberi tahu Heroku Container Stack untuk memakai `Dockerfile.heroku`
- `.dockerignore` root — mengecilkan konteks build `Dockerfile.heroku`
- Health check `/health` tersedia di `server/cmd/server/main.go`

Keduanya memakai Go 1.25, `CGO_ENABLED=0`, user non-root, dan membaca `$PORT` dari Heroku.

### 6.2 Opsi deploy

**Opsi A — container push dari lokal.** Butuh Docker Desktop dan Heroku CLI:

```sh
heroku create yeonly-api --stack container
heroku config:set ALLOWED_ORIGINS="https://nayeonworks.tech,https://www.nayeonworks.tech" -a yeonly-api
heroku container:login
cd server
heroku container:push web -a yeonly-api
heroku container:release web -a yeonly-api
heroku ps:type web=basic -a yeonly-api
heroku ps:scale web=1 -a yeonly-api
heroku domains:add api.nayeonworks.tech -a yeonly-api
heroku certs:auto -a yeonly-api
```

**Opsi B — build di Heroku lewat GitHub (tanpa Docker lokal).** Ini jalur yang dipakai bila Docker/WSL2 belum tersedia:

1. Pastikan `heroku.yml` dan `Dockerfile.heroku` sudah ada di branch `main`.
2. Heroku: **New → Create new app**, lalu **Settings → Stack → Container**.
3. **Deploy → GitHub**, hubungkan `devinayeon/yeonly`, branch `main`.
4. **Settings → Config Vars**: `ALLOWED_ORIGINS=https://nayeonworks.tech,https://www.nayeonworks.tech`.
5. **Resources**: tipe dyno **Basic**, jumlah **1**.
6. Deploy, lalu **Settings → Domains → Add domain** `api.nayeonworks.tech`.

`PORT` diisi otomatis oleh Heroku dan sudah dibaca `server/cmd/server/main.go:20`. `WEB_DIST` sengaja dibiarkan kosong agar backend berjalan API-only.

### 6.3 Konfigurasi

| Variable | Nilai | Wajib |
| --- | --- | --- |
| `PORT` | otomatis Heroku | ya |
| `WEB_DIST` | kosong (mode API-only) | tidak |
| `ALLOWED_ORIGINS` | `https://nayeonworks.tech,https://www.nayeonworks.tech` | nanti |

### 6.4 Batasan platform

- **Satu dyno saja**: state ruang hidup di memori proses; dua dyno = dua dunia terpisah.
- **Restart membuang semua ruang**: Heroku merestart dyno minimal 1×/24 jam dan setiap deploy.
- **Tidak ada persistent disk**: sesuai desain efemeral, tetapi jangan menjanjikan ruang tahan lama.
- **Biaya**: akun memiliki Platform Credits $312, sehingga biaya efektif $0 sampai saldo habis. **Pilih Basic ($7/bulan, tanpa sleep), bukan Eco ($5/bulan)**, karena Eco tidur setelah periode idle: dyno yang tidur kehilangan seluruh state ruang in-memory dan permintaan berikutnya terkena cold start. Kredit $312 cukup untuk ±44 bulan Basic (atau ±62 bulan Eco). Periksa tanggal kedaluwarsa kredit di halaman Billing; kredit promosi umumnya memiliki masa berlaku.

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
| Eco dyno tidur saat idle | Ruang in-memory hilang, cold start saat kelas dimulai | Gunakan Basic (didukung kredit $312); hindari Eco |
| CORS `*` | Origin mana pun dapat memanggil API | Sudah diganti allowlist `ALLOWED_ORIGINS`; set nilainya di Heroku |
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

- [x] Fase 0: hapus konfigurasi Firebase, perbarui `.gitignore`
- [x] Fase 1: env API/WS base, `_redirects`, uji `npm run validate` (23 tes lulus, build sukses)
- [x] Fase 2: Dockerfile, `.dockerignore`, `heroku.yml`, `Dockerfile.heroku`, dokumentasi deploy Heroku
- [ ] Fase 3: proyek Pages, env var, uji preview (menunggu akun)
- [ ] Fase 4: custom domain apex + www + `api`, verifikasi TLS (menunggu akun)
- [x] Fase 5: allowlist CORS/WS; heartbeat sudah ada
- [ ] Blocker terpisah: perbaiki identitas WebSocket (lihat bagian 14)

## 13. Perkiraan Biaya

| Layanan | Biaya |
| --- | --- |
| Cloudflare Pages | Gratis (bandwidth tak terbatas) |
| Cloudflare DNS + TLS | Gratis |
| Heroku Basic | ±$7/bulan, tanpa sleep — **dipilih** |
| Heroku Eco | ±$5/bulan, sleep saat idle — tidak dipilih |
| **Saldo Platform Credits** | **$312 → ±44 bulan Basic dengan biaya aktual $0** (cek masa berlaku) |
| Domain | sudah dimiliki |

Catatan: kredit menutup biaya dyno, tetapi tidak menghapus kebutuhan verifikasi kartu. Tetap pantau Billing agar tidak terjadi penagihan otomatis setelah kredit habis. Jika kredit mendekati kedaluwarsa, evaluasi ulang opsi Cloud Run sebelum beralih ke tagihan kartu.

## 14. Temuan Pra-Deploy: Identitas WebSocket

`go test ./...` gagal pada tiga tes **sebelum** perubahan deployment ini:

| Tes | Penyebab |
| --- | --- |
| `TestFullE2EFlow` | Klien mengirim `session=sess_...`; `JoinRoom` menolak token tak dikenal sebagai `ErrIdentity` (`room/identity.go:29-33`), padahal frontend selalu mengirim ID buatan klien (`roomStore.initSession`) |
| `TestIndependentUpvotesMultiUser` | Efek berantai koneksi pertama yang gagal |
| `TestTypingPresenceLifecycle` | Tes mengirim `hostKey` di payload, tetapi `hub.go:233-238` mewajibkan `c.IsHost`; setup tes belum diperbarui |

Dampak: pengguna baru berisiko menerima `IDENTITY_EXPIRED` pada koneksi WebSocket pertama, dan reconnect mengirim `participantId` (bukan token) sehingga identitas tidak ditemukan. Ini sudah tercatat sebagai *remaining work* di `docs/javascript-review.md:33`.

Perbaikan yang disarankan (pekerjaan terpisah, bukan bagian deployment):

1. Terbitkan token peserta dari server pada koneksi pertama, kirim lewat `ROOM_STATE`, dan simpan di `sessionStorage`.
2. Reconnect mengirim token server, bukan ID buatan klien.
3. Perbarui tes E2E dan tes typing agar memakai kontrak baru.
4. Verifikasi ulang `go test ./...` sebelum go-live.

