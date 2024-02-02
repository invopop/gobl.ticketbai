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
	cert       string
	password   string
	zone       string
	nif        string
	name       string
	year       int
	swNIF      string
	swName     string
	swVersion  string
	swLicense  string
	production bool
	page       int
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
	f.StringVar(&c.cert, "cert", "", "Certificate for authentication")
	f.StringVar(&c.password, "password", "", "Password of the certificate")
	f.StringVar(&c.nif, "nif", "", "NIF of the taxpayer")
	f.StringVar(&c.name, "name", "", "Name of the taxpayer")
	f.IntVar(&c.year, "year", 0, "Year of the invoice")
	f.StringVarP(&c.zone, "zone", "z", "", "Zone of the documents (BI, AL or GU)")
	f.StringVar(&c.swNIF, "sw-nif", "", "NIF of the software company")
	f.StringVar(&c.swName, "sw-name", "", "Name of the software")
	f.StringVar(&c.swVersion, "sw-version", "", "Version of the software")
	f.StringVar(&c.swLicense, "sw-license", "", "License of the software")
	f.BoolVarP(&c.production, "production", "p", false, "Production environment")
	f.IntVarP(&c.page, "page", "P", 1, "Page of the results")

	cmd.MarkFlagRequired("cert")             // nolint:errcheck
	cmd.MarkFlagRequired("password")         // nolint:errcheck
	cmd.MarkFlagRequired("nif")              // nolint:errcheck
	cmd.MarkFlagRequired("name")             // nolint:errcheck
	cmd.MarkFlagRequired("year")             // nolint:errcheck
	cmd.MarkFlagRequired("software-nif")     // nolint:errcheck
	cmd.MarkFlagRequired("software-name")    // nolint:errcheck
	cmd.MarkFlagRequired("software-version") // nolint:errcheck
	cmd.MarkFlagRequired("software-license") // nolint:errcheck
	cmd.MarkFlagRequired("zone")             // nolint:errcheck

	return cmd
}

func (c *fetchOpts) runE(*cobra.Command, []string) error {
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

	_, err = tbai.Fetch(l10n.Code(c.zone), c.nif, c.name, c.year, c.page)
	if err != nil {
		panic(err)
	}

	return nil
}
