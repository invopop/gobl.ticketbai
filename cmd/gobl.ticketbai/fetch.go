// Package main provides the command line interface to the TicketBAI package.
package main

import (
	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
	"github.com/spf13/cobra"
)

type fetchOpts struct {
	*rootOpts

	nif  string
	name string
	year int
	page int
}

func fetch(o *rootOpts) *fetchOpts {
	return &fetchOpts{rootOpts: o}
}

func (c *fetchOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetches issued documents from the TicketBAI provider",
		RunE:  c.runE,
	}

	f := cmd.Flags()
	c.prepareFlags(f)

	f.StringVar(&c.nif, "nif", "", "Tax ID of supplier")
	f.StringVar(&c.name, "name", "", "Name of the supplier")
	f.IntVar(&c.year, "year", 0, "Year of the invoice")
	f.IntVarP(&c.page, "page", "P", 1, "Page of the results")

	cmd.MarkFlagRequired("nif")  // nolint:errcheck
	cmd.MarkFlagRequired("name") // nolint:errcheck
	cmd.MarkFlagRequired("year") // nolint:errcheck

	return cmd
}

func (c *fetchOpts) runE(cmd *cobra.Command, _ []string) error {
	soft := c.software()

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

	tbai, err := ticketbai.New(soft, opts...)
	if err != nil {
		panic(err)
	}

	_, err = tbai.Fetch(cmd.Context(), l10n.Code(c.zone), c.nif, c.name, c.year, c.page)
	if err != nil {
		panic(err)
	}

	return nil
}
