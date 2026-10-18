# Epic PRD: Yeonly — Ephemeral Anonymous Classroom Q&A & Live Pulse

## 1. Epic Name
**Yeonly: Lightweight Ephemeral Anonymous Q&A Platform for Lectures & Seminars**

---

## 2. Goal

### Problem
In university lectures, seminars, and training sessions, many students hesitate to ask questions or express confusion due to social anxiety, fear of judgment, or embarrassment. Existing platforms either require complex account registrations, persist sensitive chat histories indefinitely, or are cluttered with distracting media (stickers, GIFs, images, voice notes) that can be abused for harassment or disrupt the focus of the class.

### Solution
**Yeonly** is an ultra-lightweight, ephemeral, text-only web application designed specifically for classroom engagement. Instructors can instantly spin up a room with an auto-generated QR code and room code. Students join anonymously without registering an account. Students can post concise questions, upvote existing questions, and provide live real-time comprehension feedback (Paham / Bingung / Terlalu Cepat). To ensure safety and zero maintenance, all data is in-memory and ephemeral (deleted upon room expiration), and multimedia is strictly disabled.

### Impact
- Eliminates the psychological barrier to asking questions, increasing student participation by >70% in quiet classes.
- Gives lecturers real-time, actionable pulse checks during presentations.
- Zero server database overhead and zero privacy liability for educational institutions.

---

## 3. User Personas

### Persona 1: The Hesitant Student ("Mahasiswa Pemalu / Introvert")
- **Profile:** 1st–4th year university student attending technical or dense lectures.
- **Pain Points:** Wants to ask for clarification on complex topics (e.g., algorithms, formulas, case studies), but fears looking foolish in front of peers and the lecturer.
- **Goals:** Submit concise questions anonymously from their smartphone browser without creating an account or revealing their identity.

### Persona 2: The Lecturer / Instructor ("Dosen / Pengajar")
- **Profile:** University lecturer or workshop facilitator presenting slides or code demos.
- **Pain Points:** Asks "Ada pertanyaan?" and is met with dead silence, only to discover later that many students failed the practical exam. Distracted by spam or unstructured chatter on WhatsApp/Telegram groups.
- **Goals:** Display a clean Q&A board on the projector, see which questions are most requested via upvotes, answer them one by one, and gauge the room's pace with a live understanding meter.

---

## 4. High-Level User Journeys

### Journey 1: Lecturer Room Creation & Projection
1. Lecturer opens `yeonly.app` on their laptop.
2. Clicks **"Buat Ruang Kelas"** (inputs optional Room Name like *"Algoritma & Pemrograman - Sesi 4"*).
3. System generates a 6-digit Room Code and Host Secret Key (saved in instructor's browser session).
4. Lecturer switches to **"Mode Proyektor"** showing a live QR code and large Room Code.
5. Lecturer projects this onto the lecture hall screen.

### Journey 2: Student Join & Anonymous Questioning
1. Student scans the QR code or navigates to `yeonly.app` and enters the 6-digit code.
2. Student is assigned a fun, temporary pseudonym (e.g., *"Pinguin Santai"*, *"Kucing Belajar"*).
3. Student views live questions submitted by peers and taps the **Upvote (▲)** button on questions they also have.
4. Student types a concise question (max 300 characters) and taps **"Kirim"**.
5. The question appears instantly on all connected devices via WebSockets.

### Journey 3: Live Pulse Check ("Understanding Meter")
1. During a difficult explanation, the lecturer asks: *"Bagaimana bagian ini?"*
2. Students click one of three quick reaction buttons:
   - 🟢 **Paham** (Understood)
   - 🟡 **Agak Bingung** (Slightly Confused)
   - 🔴 **Kecepetan / Ulangi** (Too Fast / Please Repeat)
3. The lecturer's screen displays a live percentage bar reflecting the room's collective state.

### Journey 4: Lecturer Moderation & Wrap-Up
1. Lecturer answers top-voted questions and clicks **"Tandai Terjawab"** (moves to answered tab).
2. If spam occurs, the lecturer can click **"Hapus"** or **"Kunci Pertanyaan Sementara"**.
3. At the end of class, the lecturer clicks **"Unduh Rekap (.txt/.md)"** to save unresolved questions for future review.
4. When all participants leave or after 2 hours of inactivity, the in-memory room completely dissolves.

---

## 5. Business & Technical Requirements

### Functional Requirements
1. **Ephemeral Room Management:**
   - Create room instantly with customizable room title and auto-generated 6-character alphanumeric code.
   - Host receives an administrative key to moderate the room.
   - Public/Private toggle (Public rooms appear in active lobby, Private rooms require code/QR).
   - In-memory lifecycle: auto-expiration after 2 hours of inactivity or manual room closure by host.

2. **Real-time Anonymous Text Messaging & Q&A:**
   - Full-duplex WebSocket communication between Golang backend and Svelte frontend.
   - Text-only: Strict stripping of HTML tags, scripts, markdown image syntax, and links if required.
   - Input limits: 5 to 300 characters per message.
   - Rate limiting: Max 1 message every 10 seconds per student session.
   - Questions can be upvoted by participants (1 upvote per student per question).

3. **Lecturer / Host Controls ("Dosen Mode"):**
   - Pin / Highlight question currently being discussed.
   - Mark question as answered (`Resolved`).
   - Delete inappropriate questions.
   - Toggle "Freeze Room" (prevent new submissions while lecture is ongoing).

4. **Live Lecture Pulse Meter:**
   - 3-point live understanding rating per active connection.
   - Real-time aggregate score pushed to room via WebSocket.

5. **Export & Recap:**
   - Export Q&A list to formatted Markdown/Plain Text for offline reference before room closure.

6. **Zero-Persistence Guarantee:**
   - No relational database or disk persistence for chat logs.
   - All state stored in thread-safe in-memory Go data structures (`sync.RWMutex`).

### Non-Functional Requirements
- **Performance:** WebSocket latency < 50ms for local/cloud broadcast; minimal Go binary memory footprint (< 30MB RAM for 100+ concurrent rooms).
- **Security:** XSS prevention, sanitized inputs, rate limiting against DoS/flooding, no PII (Personally Identifiable Information) collection.
- **Safety / Content Hygiene:** Disallow any media uploads (no file picker, no image pastes, no audio, no video). Profanity filter for vulgar words.
- **Accessibility & Design:** Responsive mobile-first UI with dark mode, high-contrast presentation mode, modern typography (Plus Jakarta Sans/Inter), and smooth micro-interactions.

---

## 6. Success Metrics
- **Zero Configuration Time:** Room creation to active Q&A takes < 5 seconds.
- **Student Onboarding:** Time to scan QR and submit a question is under 15 seconds.
- **Reliability:** 100% ephemeral cleanup upon timer expiration with zero memory leaks.

---

## 7. Out of Scope
- User accounts, email registration, and OAuth logins.
- Image, audio, video, GIF, or sticker uploads.
- Persistent database storage (PostgreSQL, MySQL, SQLite).
- Private 1-on-1 direct messaging between students.
- Payment gateways or monetization systems.

---

## 8. Business Value
**High:** Solves a genuine pedagogical bottleneck in Southeast Asian higher education and interactive training, offering a privacy-first, zero-liability tool with zero operational cost to run on low-resource VPS or free-tier servers.
