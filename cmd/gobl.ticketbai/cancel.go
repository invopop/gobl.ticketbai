// Package main provides the command line interface to the TicketBAI package.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/invopop/gobl"
	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
	"github.com/spf13/cobra"
)

type cancelOpts struct {
	*rootOpts
}

func cancel(o *rootOpts) *cancelOpts {
	return &cancelOpts{rootOpts: o}
}

func (c *cancelOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [infile]",
		Short: "Cancels an invoice in the TicketBAI provider",
		RunE:  c.runE,
	}

	f := cmd.Flags()
	c.prepareFlags(f)

	return cmd
}

func (c *cancelOpts) runE(cmd *cobra.Command, args []string) error {
	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(input); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	env := new(gobl.Envelope)
	if err := json.Unmarshal(buf.Bytes(), env); err != nil {
		return fmt.Errorf("unmarshaling gobl envelope: %w", err)
	}

	cert, err := xmldsig.LoadCertificate(c.cert, c.password)
	if err != nil {
		panic(err)
	}

	opts := []ticketbai.Option{
		ticketbai.WithCertificate(cert),
		ticketbai.WithZone(l10n.Code(c.zone)),
		ticketbai.WithThirdPartyIssuer(),
	}

	if c.production {
		opts = append(opts, ticketbai.InProduction())
	} else {
		opts = append(opts, ticketbai.InTesting())
	}

	tbai, err := ticketbai.New(c.software(), opts...)
	if err != nil {
		panic(err)
	}

	doc, err := tbai.NewCancelDocument(env)
	if err != nil {
		panic(err)
	}

	err = doc.Fingerprint()
	if err != nil {
		panic(err)
	}

	if err := doc.Sign(); err != nil {
		panic(err)
	}

	err = tbai.Cancel(cmd.Context(), doc)
	if err != nil {
		panic(err)
	}

	return nil
}
