// Package main rebuilds every XML fixture in test/data/out/ from the JSON
// inputs in test/data/. Run with:
//
//	go run ./cmd/regen-xmls
//
// This is a libxml2-free stand-in for `go test -update ./...` against the
// top-level package's TestXMLGeneration. It skips the XSD validation step
// (which is the only reason the test imports libxml2) and just writes the
// converted, signed XML to disk. CI should still run the full test to keep
// schema validation honest.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/gobl.ticketbai/convert"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/xmldsig"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	client, err := newClient()
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	examples, err := filepath.Glob(test.Path("test", "data", "*.json"))
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}

	for _, in := range examples {
		name := filepath.Base(in)
		out := test.Path("test", "data", "out", strings.TrimSuffix(name, ".json")+".xml")
		data, err := convertOne(client, name)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", out, err)
		}
		fmt.Println("ok:", out)
	}
	return nil
}

func newClient() (*ticketbai.Client, error) {
	pass, err := os.ReadFile(test.Path("test", "certs", "EntitateOrdezkaria_RepresentanteDeEntidad_pin.txt"))
	if err != nil {
		return nil, err
	}
	cert, err := xmldsig.LoadCertificate(
		test.Path("test", "certs", "EntitateOrdezkaria_RepresentanteDeEntidad.p12"),
		strings.TrimSpace(string(pass)),
	)
	if err != nil {
		return nil, err
	}
	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	if err != nil {
		return nil, err
	}
	return ticketbai.New(
		&ticketbai.Software{
			Licenses: ticketbai.Licenses{"sandbox": {ticketbai.ZoneBI: "My License"}},
			NIF:      "12345678A",
			Name:     "My Software",
			Version:  "1.0",
		},
		ticketbai.ZoneBI,
		ticketbai.WithCertificate(cert),
		ticketbai.WithCurrentTime(ts),
		ticketbai.WithThirdPartyIssuer(),
	)
}

func convertOne(c *ticketbai.Client, name string) ([]byte, error) {
	env := test.LoadEnvelope(name)
	td, err := c.Convert(env)
	if err != nil {
		return nil, fmt.Errorf("convert: %w", err)
	}
	if err := c.Fingerprint(td, &convert.ChainData{
		Series:    "AF",
		Code:      "1234567890",
		IssueDate: "01-01-2021",
		Signature: strings.Repeat("1234567890", 20),
	}); err != nil {
		return nil, fmt.Errorf("fingerprint: %w", err)
	}
	if err := c.Sign(td, env); err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	return td.BytesIndent()
}
