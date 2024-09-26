package ticketbai

import (
	"fmt"
	"strings"
	"time"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
)

// CancelDocument is a wrapper around the internal AnulaTicketBAI document.
type CancelDocument struct {
	env  *gobl.Envelope
	inv  *bill.Invoice
	tbai *doc.AnulaTicketBAI // output

	client *Client
}

// NewCancelDocument creates a new AnulaTicketBAI document from the provided
// GOBL Envelope.
func (c *Client) NewCancelDocument(env *gobl.Envelope) (*CancelDocument, error) {
	d := new(CancelDocument)

	// Set the client for later use
	d.client = c

	// Extract the Invoice
	var ok bool
	d.env = env
	d.inv, ok = d.env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrOnlyInvoices
	}

	if d.inv.Supplier.TaxID.Country != l10n.ES.Tax() {
		return nil, ErrNotSpanish
	}

	// Extract the time when the invoice was posted to TicketBAI gateway
	ts, err := extractPostTime(d.env)
	if err != nil {
		return nil, err
	}

	// Create the document
	d.tbai, err = doc.NewAnulaTicketBAI(d.inv, ts)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Fingerprint generates a finger print for the TicketBAI document using the
// data provided from the previous invoice data.
func (d *CancelDocument) Fingerprint() error {
	c := d.client // shortcut

	conf := &doc.FingerprintConfig{
		License:         c.software.License,
		NIF:             c.software.NIF,
		SoftwareName:    c.software.Name,
		SoftwareVersion: c.software.Version,
	}
	return d.tbai.Fingerprint(conf)
}

// Sign is used to generate the XML DSig components of the final XML document.
func (d *CancelDocument) Sign() error {
	c := d.client // shortcut

	dID := d.env.Head.UUID.String()
	if err := d.tbai.Sign(dID, c.cert, c.issuerRole, xmldsig.WithCurrentTime(c.CurrentTime)); err != nil {
		return fmt.Errorf("signing: %w", err)
	}

	return nil
}

func extractPostTime(env *gobl.Envelope) (time.Time, error) {
	for _, stamp := range env.Head.Stamps {
		if stamp.Provider == tbai.StampCode {
			parts := strings.Split(stamp.Value, "-")
			ts, err := time.Parse("020106", parts[2])
			if err != nil {
				return time.Time{}, fmt.Errorf("parsing previous invoice date: %w", err)
			}

			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("missing previous %s stamp in envelope", tbai.StampCode)
}
