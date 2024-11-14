// Package ticketbai provides a client for generating and sending TicketBAI
// documents to the different regional services in the Basque Country.
package ticketbai

import (
	"context"
	"time"

	"github.com/invopop/gobl.ticketbai/doc"
	"github.com/invopop/gobl.ticketbai/internal/gateways"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
	"github.com/nbio/xml"
)

// Expose zone codes for external use.
const (
	ZoneBI l10n.Code = doc.ZoneBI // Bizkaia
	ZoneSS l10n.Code = doc.ZoneSS // Gipuzkoa
	ZoneVI l10n.Code = doc.ZoneVI // Araba
)

// Client provides the main interface to the TicketBAI package.
type Client struct {
	software   *Software
	zone       l10n.Code
	cert       *xmldsig.Certificate
	env        gateways.Environment
	issuerRole doc.IssuerRole
	curTime    time.Time
	gw         gateways.Connection
}

// Option is used to configure the client.
type Option func(*Client)

// WithCertificate defines the signing certificate to use when producing the
// TicketBAI document.
func WithCertificate(cert *xmldsig.Certificate) Option {
	return func(c *Client) {
		c.cert = cert
	}
}

// WithCurrentTime defines the current time to use when generating the TicketBAI
// document. Useful for testing.
func WithCurrentTime(curTime time.Time) Option {
	return func(c *Client) {
		c.curTime = curTime
	}
}

// WithConnection defines a new gateway connection to use for the client.
// Useful for testing and mocking the connection responses.
func WithConnection(conn gateways.Connection) Option {
	return func(c *Client) {
		c.gw = conn
	}
}

// WithSupplierIssuer set the issuer type to supplier. To be used when the
// invoice's supplier, using their own certificate, is issuing the document.
func WithSupplierIssuer() Option {
	return func(c *Client) {
		c.issuerRole = doc.IssuerRoleSupplier
	}
}

// WithCustomerIssuer set the issuer type to customer. To be used when the
// invoice's supplier, using their own certificate, is issuing the document.
func WithCustomerIssuer() Option {
	return func(c *Client) {
		c.issuerRole = doc.IssuerRoleCustomer
	}
}

// WithThirdPartyIssuer set the issuer type to third party. To be used when the
// an authorised third party, using their own certificate, is issuing the
// document on behalf of the invoice's supplier.
func WithThirdPartyIssuer() Option {
	return func(c *Client) {
		c.issuerRole = doc.IssuerRoleThirdParty
	}
}

// InProduction defines the connection to use the production environment.
func InProduction() Option {
	return func(c *Client) {
		c.env = gateways.EnvironmentProduction
	}
}

// InTesting defines the connection to use the testing environment.
func InTesting() Option {
	return func(c *Client) {
		c.env = gateways.EnvironmentTesting
	}
}

// Software defines the details about the software that is using this library to
// generate TicketBAI documents. These details are included in the final
// document.
type Software struct {
	License     string
	NIF         string
	Name        string
	CompanyName string
	Version     string
}

// New creates a new TicketBAI client with shared software and configuration
// options for creating and sending new documents.
func New(software *Software, zone l10n.Code, opts ...Option) (*Client, error) {
	c := new(Client)
	c.software = software
	c.zone = zone

	// Set default values that can be overwritten by the options
	c.env = gateways.EnvironmentTesting
	c.issuerRole = doc.IssuerRoleSupplier

	for _, opt := range opts {
		opt(c)
	}

	var err error
	c.gw, err = gateways.New(c.env, c.zone, c.cert)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Post will send the document to the TicketBAI gateway.
func (c *Client) Post(ctx context.Context, d *doc.TicketBAI) error {
	if err := c.gw.Post(ctx, d); err != nil {
		return newErrorFrom(err)
	}
	return nil
}

// Cancel will send the cancel document in the TicketBAI gateway.
func (c *Client) Cancel(ctx context.Context, d *doc.AnulaTicketBAI) error {
	return c.gw.Cancel(ctx, d)
}

// ParseDocument will parse the XML data into a TicketBAI document.
func ParseDocument(data []byte) (*doc.TicketBAI, error) {
	d := new(doc.TicketBAI)
	if err := xml.Unmarshal(data, d); err != nil {
		return nil, err
	}
	return d, nil
}

// ParseCancelDocument will parse the XML data into a Cancel TicketBAI document.
func ParseCancelDocument(data []byte) (*doc.AnulaTicketBAI, error) {
	d := new(doc.AnulaTicketBAI)
	if err := xml.Unmarshal(data, d); err != nil {
		return nil, err
	}
	return d, nil
}

// CurrentTime returns the current time to use when generating
// the TicketBAI document.
func (c *Client) CurrentTime() time.Time {
	if !c.curTime.IsZero() {
		return c.curTime
	}
	return time.Now()
}
