package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/3cpo-dev/gaxx/internal/agent"
)

func main() {
	addr := ":8088"
	srv := &agent.Server{Version: "dev"}
	go func() {
		if err := srv.ListenAndServe(addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()
	fmt.Fprintf(os.Stdout, "gaxx-agent listening on %s\n", addr)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	fmt.Fprintln(os.Stdout, "gaxx-agent shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
