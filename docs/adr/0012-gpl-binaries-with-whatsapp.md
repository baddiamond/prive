# 0012 — Binaries with the WhatsApp engine are distributed under GPL-3.0

- **Status:** accepted
- **Date:** 2026-10-20

## Context

Privé's source code is MIT-licensed. The WhatsApp engine uses whatsmeow (MPL-2.0, [ADR 0005](0005-direct-whatsmeow.md)),
which depends on **`go.mau.fi/libsignal`**, the Go implementation of the Signal protocol, licensed under
**GPL-3.0**. [ADR 0011](0011-licensing-policy.md) requires a new ADR for any GPL dependency.

There is no GPL-free alternative: every Go implementation of the Signal protocol descends from the same GPL-3.0
code, and the official `libsignal` is AGPL-3.0. A WhatsApp client cannot work without the Signal protocol.

Go links everything into one binary, so a distributed build that includes the WhatsApp engine is a combined work
covered by the GPL-3.0.

## Decision

- The **source code in this repository stays MIT**. Every file we write can still be reused under MIT.
- **Binaries that include the WhatsApp engine** (all official releases) are distributed under the **GPL-3.0**.
  This is compatible with MIT and MPL-2.0 code, and Privé already publishes its full source.
- Releases ship `THIRD_PARTY_NOTICES.md` and the GPL-3.0 text, and Settings → About states the binary license.
- The README says this plainly in its License section.
- Engines remain separate packages: a build without the WhatsApp engine has no GPL code.

This is an engineering decision based on the licenses as published; it is not legal advice.

## Consequences

- Users get the GPL-3.0 freedoms for the app they run; nothing changes for them in practice.
- Nobody can ship a closed-source fork of a build that includes WhatsApp.
- Contributors keep contributing under MIT.
- The license check in CI (ADR 0011) allows `go.mau.fi/libsignal` explicitly and fails on any other GPL module.

## Alternatives considered

- **Relicense the whole project to GPL-3.0** — also valid, but loses MIT reuse of code that does not touch WhatsApp.
- **Run the WhatsApp engine as a separate process** (out-of-process engines, roadmap) — much more complex, and
  whether it avoids a combined work is legally uncertain.
- **Drop WhatsApp** — defeats the MVP.
