// dockupdate is a terminal UI for managing containers across Docker and
// Podman: compose-aware inventory, network browsing, and interactive
// updates with live progress.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/amagyar/dockupdate/internal/tui"
)

// version is injected at link time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		socket      = flag.String("socket", "", "engine socket (e.g. unix:///var/run/docker.sock); overrides auto-detection")
		concurrency = flag.Int("concurrency", 3, "max concurrent updates")
		prune       = flag.Bool("prune", false, "remove old images after successful updates")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println("dockupdate", version)
		return
	}
	if *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "dockupdate: --concurrency must be >= 1")
		os.Exit(2)
	}

	err := tui.Run(tui.Options{
		Socket:      *socket,
		Concurrency: *concurrency,
		Prune:       *prune,
		Version:     version,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dockupdate:", err)
		os.Exit(1)
	}
}
