# Technical Specification: Yeonly — Ephemeral Anonymous Classroom Chat & Q&A

## Problem Statement
In educational environments such as university lectures and training seminars, shy or hesitant students are reluctant to ask questions or express confusion publicly due to fear of judgment or public speaking anxiety. Traditional communication channels require login credentials, leave permanent logs that students may regret later, or become overrun by memes, stickers, media attachments, and irrelevant banter. Instructors lack a frictionless, distraction-free channel to receive real-time, anonymous student questions and gauge room comprehension during a lecture.

## Solution
Yeonly provides an instant, zero-login, text-only web platform where an instructor can create an ephemeral room in seconds, project a QR code, and allow students to submit questions anonymously. Students can vote on questions to bubble up the most critical doubts, and provide 1-click understanding feedback (Paham / Bingung / Terlalu Cepat). All room data resides purely in-memory on a high-concurrency Go backend, ensuring absolute privacy and zero storage maintenance. Media (images, audio, video, GIFs) is strictly prohibited.

---

## User Stories

1. As an instructor, I want to create a lecture room with a single click and an optional subject name, so that I can set up a session in seconds without registration.
2. As an instructor, I want to receive a 6-character room code and a private host management token, so that only I can access administrative controls for the room.
3. As an instructor, I want to view a large projector-friendly QR code and room code on screen, so that students in the lecture hall can easily join from their phones.
4. As an instructor, I want to see questions sorted by upvotes in real-time, so that I can address the most pressing student doubts first.
5. As an instructor, I want to highlight or pin the question currently being addressed, so that all students know what is being discussed.
6. As an instructor, I want to mark answered questions as resolved, so that the active Q&A queue remains focused and uncluttered.
7. As an instructor, I want to delete inappropriate or off-topic questions, so that the discussion remains respectful and productive.
8. As an instructor, I want to freeze question submissions temporarily, so that students focus on the lecture during intensive explanations.
9. As an instructor, I want to view a live aggregate Understanding Meter (Paham, Agak Bingung, Terlalu Cepat), so that I know whether to slow down or repeat a concept.
10. As an instructor, I want to export the Q&A recap as a Markdown or plain text file before ending the lecture, so that I have notes of student difficulties to address in the next class.
11. As an instructor, I want to close the room manually at the end of class, so that all in-memory data is instantly and irrevocably erased.
12. As a student, I want to join a room by typing a 6-character code or scanning a QR code, so that I don't need to create an account or provide my email.
13. As a student, I want to be assigned a friendly, random anonymous pseudonym, so that I feel safe asking questions without revealing my identity.
14. As a student, I want to submit text-only questions (5 to 300 characters), so that I can express my doubts concisely.
15. As a student, I want the system to reject any image, video, audio, or GIF files, so that the chat is safe from offensive or distracting media.
16. As a student, I want to upvote my classmates' questions, so that I can signal that I share the same confusion without posting duplicate questions.
17. As a student, I want to provide quick 1-tap feedback on lecture pace (Paham, Bingung, Kecepatan), so that the lecturer knows the class's overall comprehension in real-time.
18. As a student, I want to see real-time updates when new questions are asked or when questions are marked answered, so that I am always looking at the current state.
19. As a student, I want clear visual confirmation that my question has been submitted, so that I don't accidentally send multiple duplicates.
20. As a visitor on the landing page, I want to see a list of active public rooms (if enabled) or quickly enter a room code, so that I can navigate to my lecture with minimal friction.

---

## Implementation Decisions

### 1. Architectural Architecture & Modules
- **Monorepo / Clean Separation:**
  - `server/`: Go HTTP + WebSocket server built with Go 1.25.
  - `web/`: Svelte 5 frontend built with Vite.
  - Single binary deployment option: Go can optionally embed the built Svelte static assets (`embed.FS`) for zero-dependency deployment.

### 2. Ephemeral In-Memory State (`server/internal/room`)
- No SQL or NoSQL database.
- State is held in a thread-safe `RoomManager` with `sync.RWMutex`.
- Structure:
  - `Room`: ID, Name, HostToken, CreatedAt, LastActiveAt, IsFrozen, IsPublic, Questions map, Connections map, PulseStats.
  - `Question`: ID, RoomID, AuthorAlias, Text, Upvotes count, Voters set, Status (`unanswered`, `pinned`, `answered`), CreatedAt.
  - `PulseFeedback`: Map of connection/session ID to vote (`paham`, `bingung`, `kecepatan`).
- Background Goroutine: Periodic cleaner (every 10 minutes) purging rooms idle for > 2 hours.

### 3. Real-Time Communication Protocol
- Full-duplex WebSocket at `/ws?room={code}&token={session_token}&role={host|student}`.
- Message envelopes:
  - `ACTION_SUBMIT_QUESTION` (client -> server)
  - `ACTION_UPVOTE_QUESTION` (client -> server)
  - `ACTION_UPDATE_PULSE` (client -> server)
  - `ACTION_HOST_PIN` (host -> server)
  - `ACTION_HOST_RESOLVE` (host -> server)
  - `ACTION_HOST_DELETE` (host -> server)
  - `ACTION_HOST_TOGGLE_FREEZE` (host -> server)
  - `EVENT_ROOM_STATE` (server -> client, full initial sync)
  - `EVENT_QUESTION_ADDED` (server -> client broadcast)
  - `EVENT_QUESTION_UPDATED` (server -> client broadcast)
  - `EVENT_QUESTION_DELETED` (server -> client broadcast)
  - `EVENT_PULSE_UPDATED` (server -> client broadcast)
  - `EVENT_ROOM_FROZEN` (server -> client broadcast)

### 4. Text Safety & Anti-Abuse Controls
- Strict text sanitization: HTML tags stripped via Go `html/template` and regex cleaners.
- Markdown image syntax `![alt](url)` and audio/video embeds explicitly stripped or rejected.
- Rate limiter: Leaky bucket / token bucket per client IP/session (max 1 question per 10s).
- Indonesian & English offensive word blacklist filter replacing vulgar words with asterisks (`***`).

### 5. Frontend Design Decisions (`web/`)
- Pure Svelte components with reactive stores.
- Vanilla CSS with CSS custom properties (design tokens):
  - Dark mode by default with modern slate/neutral colors, indigo & cyan accents, subtle glassmorphism (`backdrop-filter`).
  - Projector Mode: High-contrast toggle with maximized fonts for projection.
  - No bloated CSS frameworks: lightweight, highly responsive, zero unnecessary bundle weight.
  - Accessible forms, mobile touch targets (min 48px), keyboard shortcuts (Enter to send).

---

## Testing Decisions

### Good Test Principles
- Test observable behavior and protocol compliance rather than private struct internals.
- Assert concurrency safety under simultaneous client writes (race condition testing with `-race`).
- Verify WebSocket connection life-cycles, message broadcasting, and rate-limit enforcement.

### Test Coverage Plan
1. **Go Room Engine (`server/internal/room`):**
   - Unit tests for room creation, code generation collision resistance, question submission, and upvoting concurrency.
   - Tests for question status transitions (`unanswered` -> `pinned` -> `answered`).
   - Tests for idle room garbage collection.
2. **Go WebSocket Handler (`server/internal/ws`):**
   - Integration tests with simulated WebSocket clients connecting, sending invalid payloads, hitting rate limits, and verifying broadcast isolation between different rooms.
3. **Frontend Component & Integration Tests:**
   - Svelte component tests for QuestionItem, PulseMeter, RoomControls, and text input validation.

---

## Out of Scope
- File attachments, picture uploads, avatar uploads.
- Permanent message storage or persistent database records.
- User accounts or authentication via external OAuth providers.
- Direct private messaging between users.

---

## Further Notes
- The application will be optimized to run on a $2.50/mo or free-tier container (e.g. Fly.io / Render / VPS) consuming less than 30MB of RAM while handling hundreds of students simultaneously.
