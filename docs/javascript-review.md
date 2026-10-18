# JavaScript review — 2026-10-18

Scope: frontend JavaScript, Svelte scripts, browser persistence, HTTP/WebSocket boundaries, application configuration and tests. The `modern-javascript-patterns` skill was installed and used as guidance. No backend behavior or visual redesign was included.

## Findings addressed

| Finding | Impact | Change |
| --- | --- | --- |
| `checkJs` existed without an executable check command or explicit domain types | Empty store arrays inferred as `never[]`; invalid parameters and Promise misuse were not caught by builds | Added compatible TypeScript/Svelte checking dependencies, strict configuration, shared contracts and JSDoc annotations; `npm run validate` checks, tests and builds |
| Composer treated `Promise<boolean>` as a boolean | Draft cleared and success displayed before server acceptance | Await acknowledgement; prevent duplicate sends; release pending state in `finally`; preserve rejected drafts; avoid sending during IME composition |
| Stored host map and server JSON were trusted | Malformed values could flow into typed stores and views | Validate stored credentials, HTTP responses and event payloads at the boundary; isolate malformed events within legacy batched frames |
| Duplicate identity generation and repeated storage try/catch blocks | Inconsistent behavior and harder maintenance | Shared storage and cryptographic ID helpers; retain existing stored IDs; support HTTP LAN origins without `randomUUID` |
| Store reset reused the same object; deletion retained personal vote IDs; `activeUsers || 1` discarded zero | State could become stale or display incorrect counts | Create fresh initial state, remove deleted question IDs from personal votes, use nullish defaults |
| Question filters copied arrays before calling `filter`, which already returns a new array | Avoidable allocation on every derived update | Filter directly, sort only the filtered result, verify source order remains intact |
| Message ownership depended on display aliases | Reconnected users or matching aliases could be misidentified | Compare `authorId` with the current session ID; remove obsolete public-voters fallback |
| Polling could overlap and pending requests could navigate after unmount | Stale view updates, unnecessary work and unexpected late navigation | Guard overlapping fetches, abort requests on component destruction, guard post-request navigation |
| Clipboard success preceded completion; QR rendering had duplicate triggers | False success notifications and redundant work | Await a reusable clipboard helper, handle failure, keep one reactive QR trigger |
| Dismissed toast timers and copy timers outlived their useful work | Unnecessary callbacks and harder lifecycle reasoning | Clear expiry timers on dismissal; clear copy timer on destruction |
| Unreachable duplicate lobby, starter counter, unused avatar utility and template README remained | Duplicate room-entry logic and misleading project guidance | Remove confirmed unused code; document the real project structure and commands |

The initial checker reported 175 errors and one warning, including Node test mocks. The application now checks with zero errors and warnings. Test fixtures intentionally contain partial browser objects; these are covered by runtime tests and are excluded from the browser application configuration. This does not claim the test files are statically type-clean.

## Validation

- Strict checks cover application `.js`, `.svelte` and declarations, including unused locals and parameters.
- Regression coverage includes identity changes, role restoration, navigation, independent votes, rejected submissions at the socket layer, immutable filtering, zero counts, deleted vote state, malformed storage, malformed event frames, HTTP validation, clipboard rejection, timer expiry and ID fallback.
- Browser verification against the Go server: room creation, accepted message submission, and immediate rate-limit rejection. The rejected draft remained in the composer and the error was displayed.
- Production bundle built successfully.

## Remaining work, in priority order

1. **Room termination needs a backend contract.** `HostToolbar.svelte` says the room and its data were deleted, but its handler only removes a local host key, disconnects and resets navigation. `server/internal/api/router.go` has no room-deletion endpoint. Implement authorized deletion and a room-closed event before claiming server data is gone.
2. **Participant authentication and alias persistence need one consistent server path.** The WebSocket handler accepts a session string and generates a new alias on each connection. The newer participant-token helpers in `server/internal/room/identity.go` are not wired into that handler. A server-side migration is needed; client IDs alone are not an authentication boundary.
3. **Moderation feedback should wait for server acknowledgement.** Pin/resolve/delete buttons still show optimistic success notifications even when authorization or network delivery can fail. Extend the existing request-ID acknowledgement pattern to these actions.
4. **Typing emission is disconnected from the composer.** The server, socket method and indicator exist, but the current input component does not call `ws.setTyping`. Restore input/blur/destroy wiring with a focused presence regression test.
5. **Large view files can be split by responsibility.** `LandingView.svelte` still owns the tabs, search/pagination, room creation and extensive CSS. Extract form and room-list components when those features next change. Avoid introducing a general component framework for one page.
6. **Keep wire contracts aligned.** Frontend declarations and validators are maintained alongside Go models. Add cross-language contract fixtures or schema generation as the protocol grows. Unknown events are safely ignored; invalid known events are logged rather than applied.
7. **Separate test type checking and component tests remain useful follow-ups.** A Node-specific configuration and complete browser mocks would allow strict checking of tests too. Add automated component tests for composer pending/rejection states; the current end-user rejection check was performed in a real browser.

The review retains supported Svelte legacy syntax and the unused but reusable `InteractiveHoverButton` component. A full Svelte-runes migration or design-system rewrite would add unrelated behavior changes. Static design assets were not removed merely because they were not imported by current components.
