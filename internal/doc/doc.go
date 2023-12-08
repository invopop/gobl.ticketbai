// Package doc contains the TicketBAI document structures and methods to
// generate it.
package doc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
)

// location is a shared location object set during init.
var location *time.Location

func init() {
	var err error
	location, err = time.LoadLocation("Europe/Madrid")
	if err != nil {
		panic(err)
	}
}

const (
	ticketBAINamespace = "urn:ticketbai:emision" // nolint:misspell
)

// TicketBAI contains the data needed to create a TicketBAI invoice
type TicketBAI struct {
	XMLName    xml.Name `xml:"T:TicketBai"`
	TNamespace string   `xml:"xmlns:T,attr"`

	Cabecera   *Cabecera          // Header
	Sujetos    *Sujetos           // Subjects
	Factura    *Factura           // Invoice
	HuellaTBAI *HuellaTBAI        // Fingerprint
	Signature  *xmldsig.Signature `xml:"ds:Signature,omitempty"` // XML Signature

	zone l10n.Code // copied from invoice
}

// Cabecera defines the document head with TBAI version ID.
type Cabecera struct {
	IDVersionTBAI string
}

// NewTicketBAI takes the GOBL Invoice and converts into a TicketBAI document
// ready to send to a regional API.
func NewTicketBAI(inv *bill.Invoice, ts time.Time) (*TicketBAI, error) {
	err := validateInvoice(inv)
	if err != nil {
		return nil, err
	}

	goblWithoutIncludedTaxes, err := inv.RemoveIncludedTaxes()
	if err != nil {
		return nil, err
	}

	doc := &TicketBAI{
		TNamespace: ticketBAINamespace,
		Cabecera: &Cabecera{
			IDVersionTBAI: "1.2",
		},
		Sujetos: &Sujetos{
			Emisor: newEmisor(inv.Supplier),
		},
		Factura: &Factura{
			CabeceraFactura: newCabeceraFactura(inv, ts),
			TipoDesglose:    newTipoDesglose(goblWithoutIncludedTaxes),
		},
	}

	// Add customers
	if inv.Customer != nil {
		doc.Sujetos.Destinatarios = &Destinatarios{
			IDDestinatario: []IDDestinatario{
				newDestinatario(inv.Customer),
			},
		}
	}

	// Complete invoice data
	doc.Factura.DatosFactura, err = newDatosFactura(goblWithoutIncludedTaxes)
	if err != nil {
		return nil, err
	}

	doc.zone = inv.Supplier.TaxID.Zone

	return doc, nil
}

// Fingerprint tries to generate the "HuellaTBAI" using the
// previous invoice details (if available) as a reference.
func (doc *TicketBAI) Fingerprint(conf *FingerprintConfig) error {
	return doc.buildHuellaTBAI(conf)
}

// Sign generates and assigns the XML signature to the document. It needs an
// ID to use to identify the document and a certificate to sign with.
func (doc *TicketBAI) Sign(docID string, cert *xmldsig.Certificate, opts ...xmldsig.Option) error {
	return doc.sign(docID, cert, opts...)
}

// QRCodes generates the QR codes for this invoice, but requires the Fingerprint to have been
// generated first.
func (doc *TicketBAI) QRCodes() *Codes {
	if doc.HuellaTBAI == nil {
		return nil
	}
	return doc.generateCodes(doc.zone)
}

// Bytes returns the XML document bytes
func (doc *TicketBAI) Bytes() ([]byte, error) {
	buf, err := doc.buffer(xml.Header, false)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// BytesIndent returns the idented XML document bytes
func (doc *TicketBAI) BytesIndent() ([]byte, error) {
	buf, err := doc.buffer(xml.Header, true)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (doc *TicketBAI) buffer(base string, indent bool) (*bytes.Buffer, error) {
	buf := bytes.NewBufferString(base)

	enc := xml.NewEncoder(buf)
	if indent {
		enc.Indent("", "  ")
	}

	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encoding document: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("closing encoder: %w", err)
	}

	return buf, nil
}
