# Yeonly frontend

JavaScript ES modules and Svelte components, built with Vite. The Go backend lives in `../server`.

## Development

```sh
npm ci
npm run dev
```

Run `go run ./cmd/server` from `../server` in another terminal. Vite proxies `/api` and `/ws` to port 8080. The Go server can also serve `web/dist` after a build.

## Validation

```sh
npm run validate
```

This runs strict application type checking, Node regression tests, then the production build. Individual commands are `npm run check`, `npm test`, and `npm run build`.

`jsconfig.json` checks all application `.js` and `.svelte` files with strict null checking, implicit-any checking, and unused-variable checks. `src/lib/types.d.ts` defines shared contracts, consumed with JSDoc so runtime code stays JavaScript. Node test files use partial browser mocks and fixtures; they are executed by `npm test` and excluded from the browser application type-check configuration.

TypeScript is pinned to the 6.0 minor line supported by the installed Svelte checker. Upgrade them together after validating compatibility.

## Code map

- `src/App.svelte`: application composition and browser navigation lifecycle.
- `src/lib/stores/navigation.js`: URL restoration and home navigation.
- `src/lib/stores/roomStore.js`: room state, identity, personal votes, derived question lists.
- `src/lib/stores/wsStore.js`: connection lifecycle, server events, request acknowledgements.
- `src/lib/stores/toastStore.js`: notification state and expiry cleanup.
- `src/lib/utils/`: HTTP access, protocol validation, storage, clipboard, ID generation.
- `src/lib/components/`: views and reusable presentation components.

## Conventions

- Use `const` unless reassignment is necessary. Filter into a new array before sorting; never sort store-owned arrays in place.
- Declare public parameters and component props with JSDoc. Use shared domain types for state, roles and events.
- Treat parsed JSON as `unknown` until checked. Keep network and browser-storage boundaries outside presentation components.
- Await operations before claiming success. Message submission resolves only after server acknowledgement; rejection or disconnection preserves the draft.
- Cancel requests and timers when their owner is destroyed. Polling must not overlap itself.
- Keep URL navigation consistent with the visible view. Roles and participant identity survive reload in tab-scoped storage; host keys remain in local storage.
- Current components primarily use Svelte's supported legacy syntax. Avoid mixing component-style migrations into unrelated bug fixes.

See [the JavaScript review](../docs/javascript-review.md) for implemented changes and remaining work.
