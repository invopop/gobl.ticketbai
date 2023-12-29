package ticketbai

import (
	"fmt"
	"strings"
	"time"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/xmldsig"
)

// CancelDocument is a wrapper around the internal AnulaTicketBAI document.
type CancelDocument struct {
	env  *gobl.Envelope
	inv  *bill.Invoice
	zone l10n.Code
	tbai *doc.AnulaTicketBAI // output

	client *Client
}

func newCancelDocument(c *Client, env *gobl.Envelope) (*CancelDocument, error) {
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

	if d.inv.Supplier.TaxID.Country != l10n.ES {
		return nil, ErrNotSpanish
	}

	// Set the zone for later use
	d.zone = d.inv.Supplier.TaxID.Zone
	if d.zone == "" {
		return nil, ErrInvalidZone
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
		if stamp.Provider == es.StampProviderTBAICode {
			parts := strings.Split(stamp.Value, "-")
			ts, err := time.Parse("020106", parts[2])
			if err != nil {
				return time.Time{}, fmt.Errorf("parsing previous invoice date: %w", err)
			}

			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("missing previous %s stamp in envelope", es.StampProviderTBAICode)
}
