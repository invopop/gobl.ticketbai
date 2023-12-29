// Package ticketbai provides a client for generating and sending TicketBAI
// documents to the different regional services in the Basque Country.
package ticketbai

import (
	"errors"
	"fmt"
	"time"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/internal/gateways"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
)

// Standard error responses.
var (
	ErrNotSpanish       = errors.New("only spanish invoices are supported")
	ErrAlreadyProcessed = errors.New("already processed")
	ErrOnlyInvoices     = errors.New("only invoices are supported")
	ErrInvalidZone      = errors.New("invalid zone")
)

// Client provides the main interface to the TicketBAI package.
type Client struct {
	software   *Software
	list       *gateways.List
	cert       *xmldsig.Certificate
	env        gateways.Environment
	issuerRole doc.IssuerRole
	curTime    time.Time
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

// WithGateway defines a new gateway connection to use for a specific zone. This
// option can be used multiple times to define multiple gateways. Useful for
// testing.
func WithGateway(code l10n.Code, conn gateways.Connection) Option {
	return func(c *Client) {
		if c.list == nil {
			c.list = new(gateways.List)
		}
		c.list.Register(code, conn)
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

// PreviousInvoice stores the fields from the previously generated invoice
// document that are linked to in the new document.
type PreviousInvoice struct {
	Series    string
	Code      string
	IssueDate string
	Signature string
}

// New creates a new TicketBAI client with shared software and configuration
// options for creating and sending new documents.
func New(software *Software, opts ...Option) (*Client, error) {
	c := new(Client)
	c.software = software

	// Set default values that can be overwritten by the options
	c.env = gateways.EnvironmentTesting
	c.issuerRole = doc.IssuerRoleSupplier

	for _, opt := range opts {
		opt(c)
	}

	// Create a new gateway list if none was created by the options
	if c.list == nil && c.cert != nil {
		list, err := gateways.New(c.env, c.cert)
		if err != nil {
			return nil, fmt.Errorf("creating gateway list: %w", err)
		}

		c.list = list
	}

	return c, nil
}

// NewDocument creates a new TicketBAI document from the provided GOBL Envelope.
// The envelope must contain a valid Invoice.
func (c *Client) NewDocument(env *gobl.Envelope) (*Document, error) {
	return newDocument(c, env)
}

// NewCancelDocument creates a new AnulaTicketBAI document from the provided
// GOBL Envelope.
func (c *Client) NewCancelDocument(env *gobl.Envelope) (*CancelDocument, error) {
	return newCancelDocument(c, env)
}

// Post will send the document to the TicketBAI gateway.
func (c *Client) Post(d *Document) error {
	conn := c.list.For(d.zone)
	if conn == nil {
		return fmt.Errorf("no gateway available for %s", d.zone)
	}

	p, err := d.tbai.Bytes()
	if err != nil {
		return fmt.Errorf("generating payload: %w", err)
	}

	return conn.Post(d.inv, p)
}

// Fetch will retrieve the issued documents from the TicketBAI gateway.
func (c *Client) Fetch(zone l10n.Code, nif string, name string, year int) error {
	conn := c.list.For(zone)
	if conn == nil {
		return fmt.Errorf("no gateway available for %s", zone)
	}

	return conn.Fetch(nif, name, year)
}

// Cancel will send the cancel document in the TicketBAI gateway.
func (c *Client) Cancel(d *CancelDocument) error {
	conn := c.list.For(d.zone)
	if conn == nil {
		return fmt.Errorf("no gateway available for %s", d.zone)
	}

	p, err := d.tbai.Bytes()
	if err != nil {
		return fmt.Errorf("generating payload: %w", err)
	}

	return conn.Cancel(d.inv, p)
}

// CurrentTime returns the current time to use when generating
// the TicketBAI document.
func (c *Client) CurrentTime() time.Time {
	if !c.curTime.IsZero() {
		return c.curTime
	}
	return time.Now()
}
