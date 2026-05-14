// Package main rebuilds every test envelope in test/data/ so that any manual
// edit to the document (e.g. swapping a NIF) is reflected in the envelope's
// head.dig. Run with:
//
//	go run ./cmd/regen-fixtures
//
// This tool is intentionally small and not part of the public CLI.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/test"
)

func main() {
	dir := test.Path("test", "data")
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read dir:", err)
		os.Exit(1)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := regen(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Println("ok:", path)
	}
}

func regen(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	env := new(gobl.Envelope)
	if err := json.Unmarshal(data, env); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if err := env.Calculate(); err != nil {
		return fmt.Errorf("calculate: %w", err)
	}
	if err := env.Validate(); err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	out, err := json.MarshalIndent(env, "", "\t")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}
