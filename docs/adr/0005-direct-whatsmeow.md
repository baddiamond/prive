# 0005 — whatsmeow called directly, no HTTP/gRPC layer

- **Status:** accepted
- **Date:** 2026-10-20

## Context

WAHA and its GOWS engine expose WhatsApp through HTTP/gRPC services built on whatsmeow.
Privé is a single-user desktop app that runs on the user's own machine.

## Decision

The WhatsApp engine imports **`go.mau.fi/whatsmeow`** and calls it in-process. WAHA and GOWS are used only
as references for session handling, event mapping and reconnection strategies — not as dependencies.

## Consequences

- No extra services, ports or serialization: lower latency and memory, smaller attack surface.
- We own session lifecycle and reconnection logic ourselves.
- whatsmeow is MPL-2.0; using it as an unmodified library is compatible with Privé's MIT license.

## Alternatives considered

- **Run WAHA/GOWS as a sidecar** — heavier, adds a local network service, and complicates packaging.
