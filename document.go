package ticketbai

import (
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.ticketbai/doc"
	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
)

// NewDocument creates a new TicketBAI document from the provided GOBL Envelope.
// The envelope must contain a valid Invoice.
func (c *Client) Convert(env *gobl.Envelope) (*doc.TicketBAI, error) {
	// Extract the Invoice
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, ErrValidation.withMessage("only invoices are supported")
	}
	// Check the existing stamps, we might not need to do anything
	if hasExistingStamps(env) {
		return nil, ErrDuplicate.withMessage("already has stamps")
	}
	if inv.Supplier.TaxID.Country != l10n.ES.Tax() {
		return nil, ErrValidation.withMessage("only spanish invoices are supported")
	}

	zone := zoneFor(inv)
	if zone == "" {
		return nil, ErrValidation.withMessage("invalid zone")
	}

	out, err := doc.NewTicketBAI(inv, c.CurrentTime(), c.issuerRole, zone)
	if err != nil {
		if _, ok := err.(*doc.ValidationError); ok {
			return nil, ErrValidation.withMessage(err.Error())
		}

		return nil, err
	}

	return out, nil
}

// ZoneFor determines the zone of the envelope.
func ZoneFor(env *gobl.Envelope) l10n.Code {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return ""
	}
	return zoneFor(inv)
}

// zoneFor determines the zone of the invoice.
func zoneFor(inv *bill.Invoice) l10n.Code {
	// Figure out the zone
	if inv == nil ||
		inv.Tax == nil ||
		inv.Tax.Ext == nil ||
		inv.Tax.Ext[tbai.ExtKeyRegion] == "" {
		return ""
	}
	return l10n.Code(inv.Tax.Ext[tbai.ExtKeyRegion])
}

// Fingerprint generates a fingerprint for the TicketBAI document using the
// data provided from the previous chain data. If there was no previous
// document in the chain, the parameter should be nil. The document is updated
// in place.
func (c *Client) Fingerprint(d *doc.TicketBAI, prev *doc.ChainData) error {
	soft := &doc.Software{
		License: c.software.License,
		NIF:     c.software.NIF,
		Name:    c.software.Name,
		Version: c.software.Version,
	}
	return d.Fingerprint(soft, prev)
}

// Sign is used to generate the XML DSig components of the final XML document.
// This method will also update the GOBL Envelope with the QR codes that are
// generated.
func (c *Client) Sign(d *doc.TicketBAI, env *gobl.Envelope) error {
	zone := ZoneFor(env)
	dID := env.Head.UUID.String()
	if err := d.Sign(dID, c.cert, c.issuerRole, zone, xmldsig.WithCurrentTime(d.IssueTimestamp)); err != nil {
		return fmt.Errorf("signing: %w", err)
	}

	// now generate the QR codes and add them to the envelope
	codes := d.QRCodes(zone)
	env.Head.AddStamp(
		&head.Stamp{
			Provider: tbai.StampCode,
			Value:    codes.TBAICode,
		},
	)
	env.Head.AddStamp(
		&head.Stamp{
			Provider: tbai.StampQR,
			Value:    codes.QRCode,
		},
	)
	return nil
}

func hasExistingStamps(env *gobl.Envelope) bool {
	for _, stamp := range env.Head.Stamps {
		if stamp.Provider.In(tbai.StampCode, tbai.StampQR) {
			return true
		}
	}
	return false
}
