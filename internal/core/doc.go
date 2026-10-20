// Package core is Privé's network-agnostic domain: the Engine and Session contract
// every messaging network implements, the shared types (accounts, chats, messages),
// the events sessions emit, coded errors, and the Manager that runs all accounts.
//
// core imports no engine, no storage implementation and no UI framework.
// Persistence is reached through the Repository interface and engines through EngineSource.
package core
