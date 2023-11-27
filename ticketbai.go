package ticketbai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/internal/gateways"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/xmldsig"
)

// Standard error responses.
var (
	ErrNotSpanish       = errors.New("only spanish invoices are supported")
	ErrAlreadyProcessed = errors.New("already processed")
	ErrOnlyInvoices     = errors.New("only invoices are supported")
	ErrInvalidZone      = errors.New("invalid zone")
)

// Client
type Client struct {
	software *Software
	list     *gateways.List
	cert     *xmldsig.Certificate
	env      string
	curTime  time.Time
}

// Option is used to configure the client.
type Option func(*Client)

// WithCertificate defines the signing certificate to use when
// producing the TicketBAI document.
func WithCertificate(cert *xmldsig.Certificate) Option {
	return func(c *Client) {
		c.cert = cert
	}
}

// InProduction defines the connection to use the production environment.
func InProduction() Option {
	return func(c *Client) {
		c.env = gateways.EnvProduction
	}
}

func WithCurrentTime(curTime time.Time) Option {
	return func(c *Client) {
		c.curTime = curTime
	}
}

// Software defines the details about the software that is using
// this library to generate TicketBAI documents. These details are
// included in the final document.
type Software struct {
	License string `json:"license"`
	NIF     string `json:"nif"` // Tax Code of the company that has developed the software.
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PreviousInvoices stores the fields from the previously generated
// invoice document that are linked to in the new document.
type PreviousInvoice struct {
	Series    string   `json:"series,omitempty"`
	Code      string   `json:"code"`
	IssueDate cal.Date `json:"issue_date"`
	Signature string   `json:"signature"`
}

// Document is a wrapper around the internal TicketBAI document.
type Document struct {
	env  *gobl.Envelope
	inv  *bill.Invoice
	zone l10n.Code
	tbai *doc.TicketBAI // output
}

// New creates a new TicketBAI client with shared software
// and configuration options for creating and sending new documents.
func New(software *Software, opts ...Option) (*Client, error) {
	c := new(Client)
	c.software = software
	c.env = gateways.EnvTesting
	for _, opt := range opts {
		opt(c)
	}
	var err error
	if c.list, err = gateways.New(c.env, c.cert); err != nil {
		return nil, fmt.Errorf("creating gateway list: %w", err)
	}
	return c, nil
}

func (c *Client) NewDocument(env *gobl.Envelope) (*Document, error) {
	d := new(Document)

	// Extract the Invoice
	var ok bool
	d.env = env
	d.inv, ok = d.env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrOnlyInvoices
	}

	// Check the existing stamps, we might not need to do anything
	if d.hasExistingStamps() {
		return nil, ErrAlreadyProcessed
	}
	if d.inv.Supplier.TaxID.Country != l10n.ES {
		return nil, ErrNotSpanish
	}
	d.zone = d.inv.Supplier.TaxID.Zone
	if d.zone == "" {
		return nil, ErrInvalidZone
	}

	var err error
	d.tbai, err = doc.NewTicketBAI(d.inv, c.CurrentTime())
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Post will send the document to the TicketBAI gateway.
func (c *Client) Post(ctx context.Context, d *Document) error {
	conn := c.list.For(d.zone)
	if conn == nil {
		return fmt.Errorf("no gateway available for %s", d.zone)
	}
	return conn.Post(ctx, d.inv, d.tbai)
}

// Fingerprint generates a finger print for the TicketBAI document
// using the data provided from the previous invoice data.
// If there was no previous invoice, the parameter should be nil.
func (c *Client) Fingerprint(d *Document, prev *PreviousInvoice) error {
	conf := &doc.FingerprintConfig{
		License:         c.software.License,
		NIF:             c.software.NIF,
		SoftwareName:    c.software.Name,
		SoftwareVersion: c.software.Version,
		LastSeries:      prev.Series,
		LastCode:        prev.Code,
		LastIssueDate:   prev.IssueDate,
		LastSignature:   prev.Signature,
	}
	return d.tbai.Fingerprint(conf)
}

// Sign is used to generate the XML DSig components of the final
// XML document. This method will also update the GOBL Envelope with
// the QR codes that are generated.
func (c *Client) Sign(d *Document) error {
	dID := d.env.Head.UUID.String()
	if err := d.tbai.Sign(dID, c.cert, xmldsig.WithCurrentTime(c.CurrentTime)); err != nil {
		return fmt.Errorf("signing: %w", err)
	}

	// now generate the QR codes and add them to the envelope
	codes := d.tbai.QRCodes()
	d.env.Head.AddStamp(
		&cbc.Stamp{
			Provider: es.StampProviderTBAICode,
			Value:    codes.TBAICode,
		},
	)
	d.env.Head.AddStamp(
		&cbc.Stamp{
			Provider: es.StampProviderTBAIQR,
			Value:    codes.QRCode,
		},
	)

	return nil
}

func (c *Client) CurrentTime() time.Time {
	if !c.curTime.IsZero() {
		return c.curTime
	}
	return time.Now()
}

func (d *Document) hasExistingStamps() bool {
	for _, stamp := range d.env.Head.Stamps {
		if stamp.Provider.In(es.StampProviderTBAICode, es.StampProviderTBAIQR) {
			return true
		}
	}
	return false
}

// String generates string output of the final TicketBAI document.
func (d *Document) String() (string, error) {
	return d.tbai.String()
}

// Bytes generates the byte output of the TicketBAI Document.
func (d *Document) Bytes() ([]byte, error) {
	return d.tbai.Bytes()
}
