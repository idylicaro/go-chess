// Command engine is the project's own chess engine, speaking UCI on stdin
// and stdout so it runs in any chess GUI or tournament manager.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/idylicaro/go-chess/internal/eval"
	"github.com/idylicaro/go-chess/internal/search"
	"github.com/idylicaro/go-chess/internal/uciserver"
)

// identity is what the engine answers to "uci".
var identity = uciserver.Identity{Name: "go-chess", Author: "Idyl Icaro dos Santos"}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	server := uciserver.New(identity, os.Stdout, newSearcher)
	if err := server.Run(ctx, os.Stdin); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, "engine:", err)
		os.Exit(1)
	}
}

func newSearcher(options search.Options) *search.Searcher {
	return search.New(eval.Evaluate, options)
}
