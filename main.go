package main

import (
	"app/cmd/client"
	"app/cmd/server"
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "failed", err)
		os.Exit(1)
	}
}

func run() error {
	actions := map[string]func([]string) error{
		server.Action: server.Run,
		client.Action: client.Run,
	}

	if len(os.Args) <= 1 {
		return fmt.Errorf("fcgi action [options]\n action : [%s]", keys(actions))
	}
	a, ok := actions[os.Args[1]]
	if !ok {
		return fmt.Errorf("unknown action %s : in [%s]", os.Args[1], keys(actions))
	}
	return a(os.Args[2:])
}

func keys(actions map[string]func([]string) error) string {
	keys := make([]string, 0, len(actions))
	for k := range actions {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
