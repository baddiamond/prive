// Package all enables every engine bundled with Privé. main.go imports it for its
// side effects; each blank import below runs that engine's engines.Register call.
//
// To bundle a new engine, add one line:
//
//	_ "github.com/baddiamond/prive/internal/engines/<id>"
package all

import (
	_ "github.com/baddiamond/prive/internal/engines/whatsapp" // WhatsApp (whatsmeow)
)
