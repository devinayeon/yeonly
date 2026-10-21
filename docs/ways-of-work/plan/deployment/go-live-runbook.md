# Yeonly Go-Live Runbook

Dokumen ini adalah urutan kerja yang harus diselesaikan setelah infrastruktur
Cloudflare Pages + Heroku aktif.

Status saat dokumen dibuat:

- `https://nayeonworks.tech` sudah `200`.
- `https://www.nayeonworks.tech` sudah `301` ke apex dan mempertahankan path.
- `https://api.nayeonworks.tech/health` sudah `200`.
- CORS API sudah mengembalikan origin `https://nayeonworks.tech`.
- SSL Cloudflare sudah `Full (strict)`.
- Blocker utama: kontrak identitas WebSocket belum benar.

---

## 0. Aturan main

1. Jangan mengubah record DNS `api` dari **DNS only** menjadi proxied.
2. Jangan menambahkan `https://yeonly.pages.dev` ke `ALLOWED_ORIGINS` untuk
   produksi. Uji aplikasi lewat `https://nayeonworks.tech`.
3. Heroku tetap **1 dyno**. Room disimpan di memori proses, jadi restart atau
   deploy akan menghapus room yang sedang aktif.
4. Jangan push perubahan dokumentasi saja saat auto-deploy aktif, karena akan
   memicu rebuild Heroku dan restart dyno. Gabungkan dengan commit fix berikutnya.
5. Semua uji manual dilakukan lewat domain apex, bukan `pages.dev`.

---

## 1. Selesaikan pengaturan Cloudflare

### 1.1 Edge Certificates

Cloudflare Dashboard -> zone `nayeonworks.tech` -> **SSL/TLS -> Edge Certificates**:

- **Always Use HTTPS**: ON
- **Automatic HTTPS Rewrites**: ON
- **Minimum TLS Version**: TLS 1.2
- **TLS 1.3**: ON

Jangan aktifkan **HSTS** dulu. Aktifkan setelah semua uji end-to-end hijau
(langkah 8).

### 1.2 Verifikasi

```powershell
# Apex harus redirect HTTP -> HTTPS
curl.exe -sSI http://nayeonworks.tech

# www tetap redirect ke apex
curl.exe -sSI https://www.nayeonworks.tech

# API health
curl.exe -sS https://api.nayeonworks.tech/health
```

Yang diharapkan:

- `http://nayeonworks.tech` -> `301` ke `https://nayeonworks.tech/`
- `https://www.nayeonworks.tech` -> `301` ke apex
- API health -> `{"app":"Yeonly","status":"ok"}`

Catatan: `api.nayeonworks.tech` memakai DNS-only, jadi **Always Use HTTPS**
Cloudflare tidak berlaku untuk hostname itu. Client memang harus memakai
`https://` dan `wss://`. Kalau nanti ingin memaksa HTTP API ke HTTPS, tambahkan
middleware aplikasi dan kecualikan `/health` agar health check container tidak
rusak.

---

## 2. Verifikasi environment dan deployment Pages

Cloudflare Pages -> project `yeonly` -> **Settings -> Environment variables ->
Production**:

```text
VITE_API_BASE=https://api.nayeonworks.tech
VITE_WS_BASE=wss://api.nayeonworks.tech
NODE_VERSION=22.12.0
```

Lalu:

1. Buka **Deployments**.
2. Pastikan deployment terbaru dibangun dari commit setelah `_redirects` dihapus.
3. Buka build log dan pastikan `npm run build` selesai tanpa error.
4. Buka `https://nayeonworks.tech` di browser.
5. Buka DevTools -> **Network**.
6. Pastikan request API menuju `https://api.nayeonworks.tech/...`, bukan
   `https://nayeonworks.tech/api/...`.
7. Pastikan koneksi WebSocket menuju `wss://api.nayeonworks.tech/ws`.

Kalau request masih menuju domain apex, environment variable belum masuk ke
build. Set ulang lalu lakukan **Retry deployment** atau push commit baru.

Jangan menguji dari `https://yeonly.pages.dev`; origin itu belum ada di
`ALLOWED_ORIGINS`, jadi API akan ditolak CORS.

---

## 3. Reproduksi bug identitas

Tujuan langkah ini adalah memastikan bug dipahami sebelum mengubah kode.

1. Buka `https://nayeonworks.tech`.
2. Buat room sebagai host.
3. Buka room itu di jendela incognito atau perangkat lain.
4. Masuk sebagai peserta.
5. Buka DevTools -> **Network -> WS**.
6. Lihat pesan yang diterima.

Gejala yang sekarang diperkirakan muncul:

```text
IDENTITY_EXPIRED
```

Penyebab:

- Frontend membuat `sess_...` sendiri di `web/src/lib/stores/roomStore.js`.
- Frontend mengirim nilai itu sebagai query `session` di
  `web/src/lib/stores/wsStore.js`.
- Server hanya menerima token yang pernah diterbitkan server di
  `server/internal/room/identity.go`.
- Karena token tidak ada di `r.Members`, server menutup koneksi dengan kode
  `4401`.
- Frontend reconnect setiap 3 detik dengan token yang sama, sehingga loop.

Kontrak yang benar:

| Nilai | Sifat | Dipakai untuk |
|---|---|---|
| `participantId` | publik | ID tampilan, penulis pertanyaan, pencocokan "pertanyaan saya" |
| `participantToken` | rahasia | reconnect WebSocket peserta |
| `hostKey` | rahasia | autentikasi host |

`participantId` **bukan** bearer token. Jangan pernah mengirim `participantId`
sebagai `session`.

---

## 4. Perbaiki kontrak identitas

### 4.1 Ringkasan perubahan

Frontend:

- Jangan membuat atau mengirim `sess_...` untuk koneksi peserta.
- Saat koneksi pertama, kirim `room` saja.
- Server akan mengirim `participantToken` dan `participantId` di `ROOM_STATE`.
- Simpan `participantToken` di `sessionStorage` per room.
- Saat reconnect, kirim token itu sebagai query `session`.
- Saat pengguna mengganti identitas, hapus token lama.
- Tangani `IDENTITY_EXPIRED` dan `ROOM_CLOSED` agar tidak terjadi loop.

Server:

- Tidak perlu mengubah cara `JoinRoom` menerbitkan token.
- Pertahankan perilaku: token kosong => buat peserta baru; token valid => pakai
  peserta lama; token tidak valid => `IDENTITY_EXPIRED`.
- Perbaiki test yang masih mengirim `sess_...` sembarang.

### 4.2 Tambahkan helper penyimpanan token

Di `web/src/lib/utils/storage.js` tambahkan:

```js
/** @param {'localStorage' | 'sessionStorage'} area @param {string} key */
export function removeStorage(area, key) {
  try {
    globalThis[area]?.removeItem(key);
  } catch {
    // Storage can be disabled; the in-memory identity remains usable.
  }
}
```

Di `web/src/lib/stores/roomStore.js`:

```js
import { readStorage, writeStorage, readHostKeys, removeStorage } from '../utils/storage.js';

const PARTICIPANT_TOKEN_PREFIX = 'yeonly_participant_token:';

/** @param {string} roomId */
export function getStoredParticipantToken(roomId) {
  if (!roomId) return '';
  return readStorage(
    'sessionStorage',
    `${PARTICIPANT_TOKEN_PREFIX}${roomId.toUpperCase()}`,
  ) ?? '';
}

/** @param {string} roomId @param {string} token */
export function storeParticipantToken(roomId, token) {
  if (!roomId || !token) return;
  writeStorage(
    'sessionStorage',
    `${PARTICIPANT_TOKEN_PREFIX}${roomId.toUpperCase()}`,
    token,
  );
}

/** @param {string} roomId */
export function removeStoredParticipantToken(roomId) {
  if (!roomId) return;
  removeStorage(
    'sessionStorage',
    `${PARTICIPANT_TOKEN_PREFIX}${roomId.toUpperCase()}`,
  );
}
```

Gunakan `sessionStorage`, bukan `localStorage`, agar token peserta tidak
bertahan selamanya di browser.

### 4.3 Ubah state room

Tambahkan field baru di `createInitialState()`:

```js
participantToken: '',
```

Di `newParticipantIdentity()`, hapus token lama:

```js
newParticipantIdentity: () => {
  const { roomId } = get({ subscribe });
  if (roomId) removeStoredParticipantToken(roomId);

  const sid = createId('sess_');
  writeStorage('sessionStorage', 'yeonly_session_id', sid);
  update(s => ({
    ...s,
    sessionId: sid,
    participantToken: '',
    userAlias: '',
    isHost: false,
    hostKey: '',
    votedQuestionIds: [],
  }));
  return sid;
},
```

Di `setRoomState(payload)`:

```js
const token = typeof payload.participantToken === 'string'
  ? payload.participantToken
  : '';

if (token) storeParticipantToken(payload.id, token);

update(s => ({
  ...s,
  roomId: payload.id,
  sessionId: payload.participantId || s.sessionId,
  participantToken: token,
  // field lain tetap seperti sebelumnya
}));
```

Catatan: `sessionId` tetap dipakai untuk mencocokkan `question.authorId` dengan
pemilik pertanyaan. Itu **public participant ID**, bukan token.

### 4.4 Ubah koneksi WebSocket

Di `web/src/lib/stores/wsStore.js`:

1. Import helper baru:

```js
import {
  roomStore,
  getStoredHostKey,
  storeHostKey,
  getStoredRoomRole,
  storeRoomRole,
  getStoredParticipantToken,
  removeStoredParticipantToken,
} from './roomStore.js';
```

2. Ganti `activeSessionId` dengan penanda identitas:

```js
let activeIdentityKey = '';
```

3. Di `connect()`:

```js
const storedToken = hostKey ? '' : getStoredParticipantToken(roomId);
const participantToken = hostKey
  ? ''
  : (storedToken || get(roomStore).participantToken || '');
const identityKey = hostKey
  ? `host:${hostKey}`
  : `participant:${participantToken}`;

if (
  socket
  && socket.readyState <= WebSocket.OPEN
  && activeRoomId === roomId
  && activeHostKey === hostKey
  && activeIdentityKey === identityKey
) return;

disconnect();

activeRoomId = roomId;
activeHostKey = hostKey;
activeIdentityKey = identityKey;
```

4. Bangun URL tanpa `session` untuk koneksi pertama peserta:

```js
let wsUrl = `${resolveWSOrigin(WS_BASE, window.location)}/ws?room=${encodeURIComponent(roomId)}`;

if (participantToken) {
  wsUrl += `&session=${encodeURIComponent(participantToken)}`;
}

if (hostKey) {
  wsUrl += `&hostKey=${encodeURIComponent(hostKey)}`;
}
```

Jangan memanggil `roomStore.initSession()` di `connect()` lagi. `initSession()`
boleh tetap ada untuk placeholder UI, tetapi nilainya tidak boleh dikirim
sebagai bearer token.

5. Saat menerima `ROOM_STATE`, sinkronkan penanda identitas:

```js
case 'ROOM_STATE':
  clearTyping();
  storeRoomRole(event.payload.id, Boolean(event.payload.isHost));
  roomStore.setRoomState(event.payload);
  activeIdentityKey = activeHostKey
    ? `host:${activeHostKey}`
    : `participant:${event.payload.participantToken || ''}`;
  for (const user of event.payload.typingUsers || []) receiveTyping(user);
  break;
```

Langkah ini penting. Tanpa ini, pemanggilan `connect()` berikutnya akan
menganggap token berubah dan memutus koneksi yang masih sehat.

### 4.5 Tangani identitas kedaluwarsa dan room ditutup

Masih di `handleInbound()`:

```js
case 'IDENTITY_EXPIRED': {
  if (activeHostKey) {
    shouldReconnect = false;
    toast.error('Kunci host tidak valid. Masukkan kunci host yang benar.');
    break;
  }

  const roomId = get(roomStore).roomId || activeRoomId;
  const hadToken = Boolean(getStoredParticipantToken(roomId));
  removeStoredParticipantToken(roomId);
  roomStore.update(s => ({ ...s, participantToken: '', sessionId: '' }));
  activeIdentityKey = 'participant:';

  if (!hadToken) {
    shouldReconnect = false;
    toast.error('Gagal membuat identitas peserta. Muat ulang halaman.');
    break;
  }

  toast.warning('Sesi peserta kedaluwarsa. Membuat identitas baru...');
  break;
}

case 'ROOM_CLOSED':
  shouldReconnect = false;
  toast.error('Ruang sudah ditutup oleh host.');
  break;
```

Dengan aturan ini:

- Token lama yang tidak valid dihapus sekali.
- Reconnect berikutnya tidak membawa token, sehingga server membuat identitas
  baru.
- Kalau server tetap menolak, reconnect berhenti, bukan loop.
- Kunci host yang salah tidak memicu pembuatan identitas peserta baru.

### 4.6 Perbaiki test frontend

File yang perlu diperbarui:

- `web/src/lib/stores/wsStore.test.js`
- `web/src/lib/stores/roomStore.test.js`

Hal yang harus diuji:

1. Koneksi pertama peserta **tidak** memiliki query `session`.
2. Setelah `ROOM_STATE` berisi `participantToken`, reconnect membawa token itu.
3. `participantId` tidak pernah dipakai sebagai `session`.
4. `rollNewIdentity()` menghapus token lama dan koneksi baru tidak membawa token.
5. `IDENTITY_EXPIRED` menghapus token dan reconnect sekali tanpa token.
6. `ROOM_CLOSED` menghentikan reconnect.
7. Host tetap memakai `hostKey` dan tidak mengirim token peserta.

Fake `sessionStorage` di test harus mendukung `removeItem`, misalnya:

```js
globalThis.sessionStorage = {
  getItem: key => sessions.get(key),
  setItem: (key, value) => sessions.set(key, value),
  removeItem: key => sessions.delete(key),
};
```

### 4.7 Perbaiki test server

File yang perlu diperbarui:

- `server/internal/api/e2e_test.go`
- `server/internal/ws/typing_test.go`
- Tambahkan test baru untuk kontrak identitas, misalnya
  `server/internal/api/identity_test.go`.

Perubahan utama:

1. Koneksi peserta pertama tidak boleh memakai `&session=sess_...`.
2. Baca `ROOM_STATE`, lalu ambil `participantToken` dan `participantId`.
3. Reconnect memakai `participantToken`, bukan `participantId`.
4. Test multi-user harus membuktikan bahwa dua peserta tanpa token awal
   mendapatkan token dan ID yang berbeda.
5. Test typing harus memakai client host yang benar untuk aksi `HOST_FREEZE`.

Contoh helper test:

```go
func readRoomState(t *testing.T, conn *websocket.Conn) model.RoomStatePayload {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var event model.WSEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("failed to read initial event: %v", err)
	}
	if event.Type != model.EventRoomState {
		t.Fatalf("expected ROOM_STATE, got %s", event.Type)
	}
	raw, _ := json.Marshal(event.Payload)
	var state model.RoomStatePayload
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("failed to decode room state: %v", err)
	}
	return state
}
```

Contoh test kontrak:

```go
func TestParticipantTokenContract(t *testing.T) {
	// ... buat room dan httptest server ...

	// 1. Join pertama tanpa session.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") +
		"/ws?room=" + createData.RoomID
	first, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	state := readRoomState(t, first)
	first.Close()

	if state.ParticipantToken == "" || state.ParticipantID == "" {
		t.Fatalf("server did not issue identity: %+v", state)
	}

	// 2. Reconnect dengan token -> identitas sama.
	reconnectURL := wsURL + "&session=" + url.QueryEscape(state.ParticipantToken)
	second, _, err := websocket.DefaultDialer.Dial(reconnectURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	again := readRoomState(t, second)
	second.Close()
	if again.ParticipantID != state.ParticipantID {
		t.Fatalf("reconnect changed participant: %s != %s",
			again.ParticipantID, state.ParticipantID)
	}

	// 3. participantId bukan token -> harus ditolak.
	badURL := wsURL + "&session=" + url.QueryEscape(state.ParticipantID)
	bad, _, err := websocket.DefaultDialer.Dial(badURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Close()
	_ = bad.SetReadDeadline(time.Now().Add(2 * time.Second))
	var rejected model.WSEvent
	if err := bad.ReadJSON(&rejected); err != nil {
		t.Fatal(err)
	}
	if rejected.Type != "IDENTITY_EXPIRED" {
		t.Fatalf("expected IDENTITY_EXPIRED, got %s", rejected.Type)
	}
}
```

Untuk `typing_test.go`, buat client host terpisah:

```go
host := &Client{
	RoomID:    r.ID,
	SessionID: "host-session",
	UserAlias: "Yeonly",
	IsHost:    true,
	HostKey:   key,
	Send:      make(chan []byte, 256),
}
h.registerClient(host)
drain(host)

// Ganti h.HandleAction(b, freeze) menjadi:
h.HandleAction(host, freeze)
```

Jangan melonggarkan pengecekan `c.IsHost` di `hub.go` hanya untuk membuat test
lama lulus. Pengecekan itu adalah kontrol keamanan yang benar.

---

## 5. Jalankan seluruh validasi

Server:

```powershell
cd server
go test ./...
go vet ./...
```

Frontend:

```powershell
cd web
npm ci
npm run check
npm test
npm run build
```

Semua harus hijau sebelum commit. Kalau `npm run check` mengeluarkan warning,
jangan gunakan `--fail-on-warnings` sebagai alasan untuk menurunkan kualitas;
perbaiki warning-nya.

---

## 6. Commit dan deploy

Gunakan pesan commit yang jelas, misalnya:

```text
fix: gunakan token peserta dari server untuk identitas WebSocket
```

Lalu:

```powershell
git status
git add server web docs
git commit -m "fix: gunakan token peserta dari server untuk identitas WebSocket"
git push origin main
```

Setelah push:

1. Cloudflare Pages membangun ulang frontend.
2. Heroku membangun ulang container dan merilis versi baru.
3. Pantau:

```powershell
heroku releases -a yeonly-api
heroku logs --tail -a yeonly-api
```

Ingat: deploy akan me-restart dyno dan menghapus room yang sedang aktif.
Lakukan saat tidak ada kelas yang sedang berjalan.

Kalau build gagal:

- Heroku: rollback ke release sebelumnya lewat dashboard atau
  `heroku releases:rollback <version> -a yeonly-api`.
- Pages: pilih deployment terakhir yang sehat lalu **Rollback**.

Jangan mematikan auto-deploy selama `main` stabil. Kalau `main` sedang
eksperimental, matikan sementara atau gunakan manual deploy.

---

## 7. Uji end-to-end setelah deploy

Gunakan `https://nayeonworks.tech`, bukan `pages.dev`.

### 7.1 Host

- [ ] Buat room baru.
- [ ] Salin kode room.
- [ ] Buka room sebagai host.
- [ ] Pastikan nama host `Yeonly` tampil.
- [ ] Pastikan QR/link dapat dibuka.

### 7.2 Peserta

- [ ] Buka room di incognito atau perangkat lain.
- [ ] Masuk sebagai peserta.
- [ ] Pastikan alias acak muncul.
- [ ] Kirim pertanyaan.
- [ ] Lakukan upvote.
- [ ] Kirim pulse `paham` / `bingung` / `kecepatan`.
- [ ] Pastikan indikator typing muncul di host.
- [ ] Refresh halaman peserta.
- [ ] Pastikan alias dan identitas tetap sama.
- [ ] Pastikan pertanyaan yang sudah di-upvote tetap ditandai.

### 7.3 Host actions

- [ ] Pin pertanyaan.
- [ ] Tandai pertanyaan sebagai terjawab.
- [ ] Hapus pertanyaan.
- [ ] Freeze room.
- [ ] Pastikan peserta tidak bisa mengirim pertanyaan saat freeze.
- [ ] Unfreeze room.
- [ ] Clear pulse.
- [ ] Unduh rekap Markdown.

### 7.4 Multi-user

- [ ] Dua peserta berbeda memberikan upvote.
- [ ] Upvote bertambah dua, bukan satu.
- [ ] Salah satu peserta membatalkan upvote.
- [ ] Upvote peserta lain tidak ikut hilang.
- [ ] `activeUsers` sesuai jumlah koneksi.

### 7.5 Skenario gagal

- [ ] Room ditutup host -> peserta melihat pesan dan reconnect berhenti.
- [ ] Token peserta dirusak di `sessionStorage` -> peserta mendapat identitas
      baru, bukan loop.
- [ ] Kunci host salah -> host berhenti reconnect dan diminta memasukkan kunci
      yang benar.
- [ ] API mati sementara -> WebSocket mencoba reconnect, lalu pulih.
- [ ] Refresh host -> host tetap host.
- [ ] Buka `/room/KODE` langsung -> SPA tetap `200`.

### 7.6 Keamanan dasar

- [ ] Peserta biasa tidak bisa mengirim aksi `HOST_*`.
- [ ] `participantToken` tidak ditampilkan di UI.
- [ ] `hostKey` tidak muncul di UI publik.
- [ ] CORS hanya mengizinkan apex dan `www`.
- [ ] `api.nayeonworks.tech` tetap DNS-only.

---

## 8. HSTS dan hardening setelah stabil

Tunggu minimal 24 jam setelah semua uji hijau.

Cloudflare -> **SSL/TLS -> Edge Certificates -> HSTS**:

- Enable HSTS: ON
- Max Age: 6 bulan
- Include subdomains: ON
- Preload: **OFF**

Jangan aktifkan preload. Preload sulit dibatalkan dan tidak diperlukan untuk
deployment ini.

Hardening opsional yang aman untuk dicoba:

1. Tambahkan security headers untuk halaman statis lewat `web/public/_headers`,
   lalu uji ulang karena Cloudflare Pages akan menerapkannya ke semua aset.
2. Tambahkan header `X-Content-Type-Options: nosniff` dan
   `Referrer-Policy: strict-origin-when-cross-origin` jika belum ada.
3. Jalankan audit dependensi:

```powershell
cd web
npm audit

cd ..\server
go vet ./...
```

4. Tindak lanjuti temuan di
   `C:\Users\Delion\security-audit-skill\yeonly\run-1\NEEDS-VALIDATION.md`.
   Prioritaskan autentikasi host, rate limit, validasi input, dan eksposur token.

Jangan menambahkan CSP ketat tanpa menguji Svelte/Vite terlebih dahulu; inline
style atau script bisa rusak.

---

## 9. Operasional setelah go-live

### Monitoring

- Pasang uptime monitor untuk `https://nayeonworks.tech` dan
  `https://api.nayeonworks.tech/health`.
- Alert ke email/Telegram.
- Pantau `heroku logs --tail -a yeonly-api` saat jam kuliah.

### Kapasitas

- Tetap 1 dyno Basic.
- Jangan scale ke lebih dari 1 dyno selama room masih in-memory.
- Kalau nanti butuh multi-dyno, pindahkan state room ke Redis/Postgres.

### Restart dan kehilangan room

- Deploy, restart dyno, atau crash akan menghapus semua room.
- Umumkan ke pengguna bahwa room bersifat sementara.
- Untuk kelas besar, jangan deploy saat sesi berlangsung.

### Biaya

- Cloudflare Pages: gratis untuk kebutuhan ini.
- Heroku Basic: sekitar $7/bulan.
- Domain: biaya tahunan registrar.

### Rollback

- Heroku: rollback release.
- Pages: rollback deployment.
- DNS: jangan mengubah record `api` saat rollback; yang berubah hanya versi
  aplikasi.

---

## 10. Checklist final

### Infrastruktur

- [ ] Apex `200`
- [ ] `www` `301` ke apex, path dan query dipertahankan
- [ ] `/room/TEST01` `200`
- [ ] API health `200`
- [ ] CORS apex benar
- [ ] API tanpa `CF-RAY` (DNS-only)
- [ ] SSL Full (strict)
- [ ] Always Use HTTPS ON
- [ ] Minimum TLS 1.2
- [ ] TLS 1.3 ON
- [ ] HSTS setelah stabil

### Aplikasi

- [ ] Kontrak `participantId` vs `participantToken` diperbaiki
- [ ] Koneksi pertama peserta tanpa `session`
- [ ] Reconnect memakai `participantToken`
- [ ] `IDENTITY_EXPIRED` tidak menyebabkan loop
- [ ] `ROOM_CLOSED` tidak menyebabkan loop
- [ ] Host memakai `hostKey`
- [ ] Test frontend hijau
- [ ] `go test ./...` hijau
- [ ] `npm run validate` hijau

### Uji manual

- [ ] Host membuat room
- [ ] Peserta join
- [ ] Pertanyaan, upvote, pulse, typing
- [ ] Pin, resolve, delete, freeze, clear pulse
- [ ] Rekap Markdown
- [ ] Refresh peserta dan host
- [ ] Multi-user upvote independen
- [ ] Mobile/QR
- [ ] Room ditutup
- [ ] Token rusak
- [ ] Kunci host salah

### Rilis

- [ ] Commit dengan pesan Indonesia yang jelas
- [ ] Tidak ada kredensial, `.env`, `dist`, `node_modules`, atau artefak lokal
- [ ] `.agents/` dan `skills-lock.json` tetap diabaikan
- [ ] Deploy sukses
- [ ] Smoke test pasca-deploy
- [ ] Monitoring aktif
- [ ] Dokumentasi deployment diperbarui
