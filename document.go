package ticketbai

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/xmldsig"
)

// Document is a wrapper around the internal TicketBAI document.
type Document struct {
	env  *gobl.Envelope
	inv  *bill.Invoice
	zone l10n.Code
	tbai *doc.TicketBAI // output

	client *Client
}

func newDocument(c *Client, env *gobl.Envelope) (*Document, error) {
	d := new(Document)

	// Set the client for later use
	d.client = c

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
	d.tbai, err = doc.NewTicketBAI(d.inv, c.CurrentTime(), c.issuerRole)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Fingerprint generates a finger print for the TicketBAI document using the
// data provided from the previous invoice data. If there was no previous
// invoice, the parameter should be nil.
func (d *Document) Fingerprint(prev *PreviousInvoice) error {
	c := d.client // shortcut

	conf := &doc.FingerprintConfig{
		License:         c.software.License,
		NIF:             c.software.NIF,
		SoftwareName:    c.software.Name,
		SoftwareVersion: c.software.Version,
	}

	if prev != nil {
		conf.LastSeries = prev.Series
		conf.LastCode = prev.Code
		conf.LastIssueDate = prev.IssueDate
		conf.LastSignature = prev.Signature
	}

	return d.tbai.Fingerprint(conf)
}

// Sign is used to generate the XML DSig components of the final XML document.
// This method will also update the GOBL Envelope with the QR codes that are
// generated.
func (d *Document) Sign() error {
	c := d.client // shortcut

	dID := d.env.Head.UUID.String()
	if err := d.tbai.Sign(dID, c.cert, c.issuerRole, xmldsig.WithCurrentTime(c.CurrentTime)); err != nil {
		return fmt.Errorf("signing: %w", err)
	}

	// now generate the QR codes and add them to the envelope
	codes := d.tbai.QRCodes()
	d.env.Head.AddStamp(
		&head.Stamp{
			Provider: es.StampProviderTBAICode,
			Value:    codes.TBAICode,
		},
	)
	d.env.Head.AddStamp(
		&head.Stamp{
			Provider: es.StampProviderTBAIQR,
			Value:    codes.QRCode,
		},
	)

	return nil
}

// Bytes generates the byte output of the TicketBAI Document.
func (d *Document) Bytes() ([]byte, error) {
	return d.tbai.Bytes()
}

// BytesIndent generates the indented byte output of the TicketBAI Document.
func (d *Document) BytesIndent() ([]byte, error) {
	return d.tbai.BytesIndent()
}

// Head returns the CabeceraFactura from the TicketBAI document.
func (d *Document) Head() *doc.CabeceraFactura {
	return d.tbai.Factura.CabeceraFactura
}

// SignatureValue provides quick access to the XML signatures final value.
func (d *Document) SignatureValue() string {
	return d.tbai.SignatureValue()
}

func (d *Document) hasExistingStamps() bool {
	for _, stamp := range d.env.Head.Stamps {
		if stamp.Provider.In(es.StampProviderTBAICode, es.StampProviderTBAIQR) {
			return true
		}
	}
	return false
}
