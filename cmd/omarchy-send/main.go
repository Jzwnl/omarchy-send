// Command omarchy-send is a LocalSend-compatible file-transfer client with a terminal
// UI, designed to run headless over SSH on Arch/Omarchy servers.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"omarchy-send/internal/app"
	"omarchy-send/internal/client"
	"omarchy-send/internal/config"
	"omarchy-send/internal/discovery"
	"omarchy-send/internal/server"
	"omarchy-send/internal/tui"
)

// controller adapts the discovery + sender + server services to tui.Controller.
type controller struct {
	disc   *discovery.Discoverer
	sender *client.Sender
	srv    *server.Server
}

func (c controller) Announce()                                         { c.disc.Announce() }
func (c controller) Send(p discovery.Peer, paths []string, pin string) { c.sender.Send(p, paths, pin) }
func (c controller) SetAutoAccept(v bool)                              { c.srv.SetAutoAccept(v) }
func (c controller) SetPIN(pin string)                                 { c.srv.SetPIN(pin) }
func (c controller) SetReceiveDir(dir string)                          { c.srv.SetReceiveDir(dir) }

// SetAlias updates the alias across all services and re-announces it.
func (c controller) SetAlias(alias string) {
	c.disc.SetAlias(alias)
	c.srv.SetAlias(alias)
	c.sender.SetAlias(alias)
	c.disc.Announce()
}

func main() {
	var (
		aliasFlag = flag.String("alias", "", "device alias (overrides config for this run)")
		portFlag  = flag.Int("port", 0, "listen port (overrides config for this run)")
		dirFlag   = flag.String("dir", "", "receive directory (overrides config for this run)")
		pinFlag   = flag.String("pin", "", "require this PIN from senders (overrides config)")
		autoFlag  = flag.Bool("auto-accept", false, "auto-accept incoming transfers (no prompt)")
		noIcons   = flag.Bool("no-icons", false, "hide Nerd Font device icons (for non-Nerd-Font terminals)")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if *aliasFlag != "" {
		cfg.Alias = *aliasFlag
		cfg.DeviceModel = *aliasFlag
	}
	if *portFlag != 0 {
		cfg.Port = *portFlag
	}
	if *dirFlag != "" {
		cfg.ReceiveDir = *dirFlag
	}
	if *pinFlag != "" {
		cfg.PIN = *pinFlag
	}
	if *autoFlag {
		cfg.AutoAccept = true
	}
	if *noIcons {
		cfg.NoIcons = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	disc := discovery.New(cfg.DeviceInfo())

	var cert *tls.Certificate
	if cfg.Protocol == "https" {
		c, err := cfg.TLSCertificate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tls: %v\n", err)
			os.Exit(1)
		}
		cert = &c
	}

	srv := server.New(server.Options{
		Info:       cfg.DeviceInfo(),
		OnPeer:     disc.NotePeer,
		Cert:       cert,
		ReceiveDir: cfg.ReceiveDir,
		AutoAccept: cfg.AutoAccept,
		PIN:        cfg.PIN,
	})
	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	if err := disc.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "discovery: %v\n", err)
		os.Exit(1)
	}

	sender := client.New(cfg.DeviceInfo())
	ctrl := controller{disc: disc, sender: sender, srv: srv}

	p := tea.NewProgram(tui.New(cfg, ctrl), tea.WithAltScreen())
	app.BridgeDiscovery(ctx, disc.Events(), p.Send)
	app.BridgeServer(ctx, srv.Accepts(), srv.Transfers(), p.Send)
	app.BridgeTransfers(ctx, sender.Events(), p.Send)
	disc.Announce() // announce immediately so we appear without waiting a tick

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
