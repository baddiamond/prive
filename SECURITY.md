# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for security problems.
Report them privately through [GitHub Security Advisories](https://github.com/baddiamond/prive/security/advisories/new).

Include what you found, how to reproduce it, and its impact. We aim to acknowledge reports within
7 days and to keep you informed until a fix is released. We are happy to credit you, unless you prefer
to stay anonymous.

## Scope

In scope:

- The Privé desktop app and its code in this repository.
- Leaks of session keys, messages or personal data from the local data folder to anywhere else.
- Injection through message content (for example, HTML or script rendered in the UI).
- Supply-chain issues in our build and release process.

Out of scope:

- Vulnerabilities in the messaging networks themselves — report those to the network.
- Attacks that require full control of the user's unlocked computer.
- Account bans by a network (see [DISCLAIMER.md](DISCLAIMER.md)).

## Our practices

- Message content is always rendered as text, never as HTML.
- No remote code, no remote configuration, no telemetry endpoints.
- Dependencies are kept minimal and reviewed; lockfiles are committed.
- Secrets never appear in logs, tests or fixtures.

## Supported versions

Until 1.0, only the latest release receives security fixes.
