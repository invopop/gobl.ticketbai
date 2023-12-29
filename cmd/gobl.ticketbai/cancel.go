// Package main provides the command line interface to the TicketBAI package.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/invopop/gobl"
	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/xmldsig"
	"github.com/spf13/cobra"
)

type cancelOpts struct {
	*rootOpts
	cert       string
	password   string
	swNIF      string
	swName     string
	swVersion  string
	swLicense  string
	production bool
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
	f.StringVar(&c.cert, "cert", "", "Certificate for authentication")
	f.StringVar(&c.password, "password", "", "Password of the certificate")
	f.StringVar(&c.swNIF, "sw-nif", "", "NIF of the software company")
	f.StringVar(&c.swName, "sw-name", "", "Name of the software")
	f.StringVar(&c.swVersion, "sw-version", "", "Version of the software")
	f.StringVar(&c.swLicense, "sw-license", "", "License of the software")
	f.BoolVarP(&c.production, "production", "p", false, "Production environment")

	cmd.MarkFlagRequired("cert")             // nolint:errcheck
	cmd.MarkFlagRequired("password")         // nolint:errcheck
	cmd.MarkFlagRequired("software-nif")     // nolint:errcheck
	cmd.MarkFlagRequired("software-name")    // nolint:errcheck
	cmd.MarkFlagRequired("software-version") // nolint:errcheck
	cmd.MarkFlagRequired("software-license") // nolint:errcheck
	cmd.MarkFlagRequired("zone")             // nolint:errcheck

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

	soft := &ticketbai.Software{
		NIF:     c.swNIF,
		Name:    c.swName,
		Version: c.swVersion,
		License: c.swLicense,
	}

	cert, err := xmldsig.LoadCertificate(c.cert, c.password)
	if err != nil {
		panic(err)
	}

	opts := []ticketbai.Option{
		ticketbai.WithCertificate(cert),
		ticketbai.WithThirdPartyIssuer(),
	}

	if c.production {
		opts = append(opts, ticketbai.InProduction())
	} else {
		opts = append(opts, ticketbai.InTesting())
	}

	tbai, err := ticketbai.New(soft, opts...)
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

	err = tbai.Cancel(doc)
	if err != nil {
		panic(err)
	}

	return nil
}
