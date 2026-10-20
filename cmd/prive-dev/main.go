// Command prive-dev is a developer tool that drives Privé's backend from the terminal:
// link WhatsApp accounts (QR code printed in the terminal), list accounts with their
// connection state, and remove them. It uses the same encrypted data as the app.
//
// Quit the Privé app before using it: two processes must not run the same sessions.
//
//	go run ./cmd/prive-dev accounts
//	go run ./cmd/prive-dev link [name] [color]
//	go run ./cmd/prive-dev remove <account-id-prefix>
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/baddiamond/prive/internal/bootstrap"
	"github.com/baddiamond/prive/internal/core"
	"github.com/baddiamond/prive/internal/engines"
	_ "github.com/baddiamond/prive/internal/engines/all" // registers bundled engines
	"github.com/baddiamond/prive/internal/keys"
)

func main() {
	dataDir := flag.String("data", "", "data folder (default: the app's data folder)")
	verbose := flag.Bool("v", false, "print backend logs")
	flag.Usage = usage
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, *dataDir, *verbose, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: prive-dev [-data dir] [-v] <command>

commands:
  accounts                 list accounts and their connection state (default)
  link [name] [color]      link a WhatsApp account; color: green amber rose sky violet coral
  remove <id-prefix>       log out and delete an account

Quit the Privé app first: two processes must not run the same sessions.`)
}

// devTool is one running backend.
type devTool struct {
	local   *bootstrap.Local
	manager *core.Manager
	events  chan core.Event
}

func run(ctx context.Context, dataDir string, verbose bool, args []string) error {
	logOut := io.Discard
	if verbose {
		logOut = os.Stderr
	}
	logger := slog.New(slog.NewTextHandler(logOut, nil))

	var (
		local *bootstrap.Local
		err   error
	)
	if dataDir != "" {
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return err
		}
		local, err = bootstrap.OpenAt(ctx, dataDir, keys.OSKeychain())
	} else {
		local, _, err = bootstrap.Open(ctx, keys.OSKeychain())
	}
	if err != nil {
		return err
	}
	defer local.Close()

	t := &devTool{local: local, events: make(chan core.Event, 256)}
	t.manager = core.NewManager(local.ManagerConfig(engines.Default(), logger, func(e core.Event) {
		select {
		case t.events <- e:
		default: // the tool only needs recent events
		}
	}))
	defer t.manager.Close()
	if err := t.manager.Start(ctx); err != nil {
		return err
	}

	cmd := "accounts"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "accounts":
		return t.accounts(ctx)
	case "link":
		return t.link(ctx, args)
	case "remove":
		if len(args) != 1 {
			return errors.New("remove needs an account id prefix")
		}
		return t.remove(ctx, args[0])
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// accounts waits for connections to settle and prints every account.
func (t *devTool) accounts(ctx context.Context) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		settled := true
		for _, s := range t.manager.Accounts() {
			if s.State == core.StateConnecting || s.State == core.StateDisconnected {
				settled = false
			}
		}
		if settled {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	list := t.manager.Accounts()
	if len(list) == 0 {
		fmt.Println("No accounts. Link one with: prive-dev link")
		return nil
	}
	fmt.Printf("%-10s %-14s %-10s %-10s %-14s %s\n", "ID", "NAME", "NETWORK", "COLOR", "PHONE", "STATE")
	for _, s := range list {
		state := string(s.State)
		if s.Code != "" {
			state += " (" + s.Code + ")"
		}
		fmt.Printf("%-10s %-14s %-10s %-10s %-14s %s\n", short(s.Account.ID), s.Account.DisplayName,
			s.Manifest.Name, s.Account.Color, maskRemote(s.Account.RemoteID), state)
	}
	return nil
}

// link adds a WhatsApp account and shows QR codes until it is linked.
func (t *devTool) link(ctx context.Context, args []string) error {
	n := len(t.manager.Accounts())
	name := fmt.Sprintf("WhatsApp %d", n+1)
	colors := []string{"green", "amber", "rose", "sky", "violet", "coral"}
	color := colors[n%len(colors)]
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		color = args[1]
	}

	acc, err := t.manager.AddAccount(ctx, core.NewAccount{EngineID: "whatsapp", DisplayName: name, Color: color})
	if err != nil {
		return err
	}
	linked := false
	defer func() {
		if !linked { // like CancelLogin in the app
			_ = t.manager.RemoveAccount(context.Background(), acc.ID)
		}
	}()
	if err := t.manager.Login(ctx, acc.ID, core.LoginQR); err != nil {
		return err
	}

	fmt.Println("On your phone: WhatsApp → Settings → Linked devices → Link a device, then scan:")
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nCanceled; the account was not added.")
			return nil
		case ev := <-t.events:
			switch e := ev.(type) {
			case core.LoginQREvent:
				if e.AccountID == acc.ID {
					fmt.Print("\033[2J\033[H") // clear the screen for the fresh code
					fmt.Println("Scan with WhatsApp → Linked devices → Link a device (code refreshes automatically):")
					printQR(os.Stdout, e.Code)
				}
			case core.LoginFailedEvent:
				if e.AccountID == acc.ID {
					return fmt.Errorf("linking failed: %s", e.Code)
				}
			case core.LoginSuccessEvent:
				if e.AccountID == acc.ID {
					linked = true
					fmt.Printf("Linked %q (%s). Waiting for the connection…\n", name, maskRemote(e.RemoteID))
				}
			case core.AccountStateEvent:
				if e.AccountID == acc.ID && linked && e.State == core.StateConnected {
					fmt.Println("Connected. Run `prive-dev accounts` after restarting to check it reconnects.")
					return nil
				}
			}
		}
	}
}

func (t *devTool) remove(ctx context.Context, prefix string) error {
	for _, s := range t.manager.Accounts() {
		if strings.HasPrefix(string(s.Account.ID), prefix) {
			err := t.manager.RemoveAccount(ctx, s.Account.ID)
			if errors.Is(err, core.ErrAccountNotFound) {
				return err
			}
			fmt.Printf("Removed %q.\n", s.Account.DisplayName)
			return nil
		}
	}
	return fmt.Errorf("no account id starts with %q", prefix)
}

func short(id core.AccountID) string {
	if len(id) > 8 {
		return string(id[:8])
	}
	return string(id)
}

// maskRemote shows only the last two digits of a phone-based identity.
func maskRemote(remote string) string {
	user, _, _ := strings.Cut(remote, "@")
	if len(user) < 2 {
		return "—"
	}
	return "•••• ••" + user[len(user)-2:]
}
