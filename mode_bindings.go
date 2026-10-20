//go:build bindings

package main

// generatingBindings is true when `wails build` / `wails generate module` runs the app only
// to read its bound methods. Local data must not be touched then.
const generatingBindings = true
