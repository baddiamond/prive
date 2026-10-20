# Contributing to Privé

Thanks for your interest! Privé is built to be easy to understand and extend — by people and by AI assistants.

## Ground rules

1. **English everywhere** in the repository (code, comments, docs, commits). Translations are the only exception.
2. **No hardcoded UI text.** Use i18n keys and update every locale file (`frontend/src/locales/*.json`).
3. **Privacy is non-negotiable.** No telemetry, analytics, remote logging or third-party services.
4. **Keep it light.** New dependencies need a short justification in the PR description.
5. **Multi-account always.** Never assume a single account per network.

The complete rules are in [`CLAUDE.md`](CLAUDE.md).

## Development setup

See [Installation](README.md#installation-development) in the README. In short:

```bash
wails doctor   # verify your toolchain
wails dev      # run with hot reload
```

## Before you open a pull request

```bash
gofmt -l main.go app.go internal   # must print nothing
go vet ./...
go test ./...
cd frontend
npm run lint
npm run typecheck   # needs the Wails bindings: run `wails dev` or `wails generate module` once
```

On Linux with WebKitGTK 4.1 only (e.g. Ubuntu 24.04), add `-tags webkit2_41` to the Go commands.
CI runs the same checks (`.github/workflows/ci.yml`).

- Add or update tests for behavior you change.
- Update docs when you change behavior, architecture or the engine contract.
- For a design decision with trade-offs, add an ADR in `docs/adr/` (copy `0000-template.md`).

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(whatsapp): show QR code during pairing
fix(store): keep unread counter when chat is archived
docs: explain engine capabilities
refactor(core): split manager event fan-out
```

Scopes: `core`, `store`, `engines`, `<engine id>`, `ui`, `i18n`, `build`, `docs`.

## Code style

**Go**
- `gofmt` formatting, idiomatic Go, errors wrapped with context (`fmt.Errorf("load chats: %w", err)`).
- Every exported identifier has a doc comment.
- Errors meant for the user are `core.CodedError` values with a translation key, never free text.
- Use `log/slog` for logging. Never log message content, phone numbers or tokens.

**TypeScript / React**
- Function components and hooks, strict TypeScript, no `any`.
- Styling with Tailwind utilities from the brand preset (`bg-surface`, `text-text-secondary`…);
  no hardcoded colors. Support both light and dark themes.
- Follow the accessibility rules in [`docs/brand/BRAND.md`](docs/brand/BRAND.md#8-accessibility).
- All strings via `useTranslation()`.

## Ways to help

- 🌍 Translate Privé into your language — [`docs/I18N.md`](docs/I18N.md)
- 🎨 Improve the brand assets or accessibility — [`docs/brand/BRAND.md`](docs/brand/BRAND.md)
- 🔌 Build a new engine — [`docs/ADDING_AN_ENGINE.md`](docs/ADDING_AN_ENGINE.md)
- 🐛 Report bugs or propose features with the issue templates
- 📖 Improve docs

## Working with AI assistants

AI-assisted contributions are welcome. Agents should read `CLAUDE.md` / `AGENTS.md` first.
As the human submitting the PR, you are responsible for reviewing and understanding the code.

## Code of conduct

Be kind, assume good intent, and keep discussions focused on the work.
