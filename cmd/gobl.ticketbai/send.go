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

type sendOpts struct {
	*rootOpts

	previous string
}

func send(o *rootOpts) *sendOpts {
	return &sendOpts{rootOpts: o}
}

func (c *sendOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [infile]",
		Short: "Sends the GOBL invoice to the TicketBAI service",
		RunE:  c.runE,
	}

	f := cmd.Flags()
	c.prepareFlags(f)

	f.StringVar(&c.previous, "prev", "", "Previous document fingerprint to chain with")

	return cmd
}

func (c *sendOpts) runE(cmd *cobra.Command, args []string) error {
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

	doc, err := tbai.NewDocument(env)
	if err != nil {
		panic(err)
	}

	var prev *ticketbai.PreviousInvoice
	if c.previous != "" {
		prev = new(ticketbai.PreviousInvoice)
		if err := json.Unmarshal([]byte(c.previous), prev); err != nil {
			panic(err)
		}
	}

	err = doc.Fingerprint(prev)
	if err != nil {
		panic(err)
	}

	if err := doc.Sign(); err != nil {
		panic(err)
	}

	err = tbai.Post(cmd.Context(), doc)
	if err != nil {
		panic(err)
	}

	np := &ticketbai.PreviousInvoice{
		Series:    doc.Head().SerieFactura,
		Code:      doc.Head().NumFactura,
		IssueDate: doc.Head().FechaExpedicionFactura,
		Signature: doc.SignatureValue(),
	}
	data, err := json.Marshal(np)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated document with fingerprint: \n%s\n", string(data))

	return nil
}
