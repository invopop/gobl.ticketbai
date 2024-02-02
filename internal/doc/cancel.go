package doc

import (
	"encoding/xml"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
)

// AnulaTicketBAI contains the data needed to cancel a TicketBAI invoice
type AnulaTicketBAI struct {
	XMLName    xml.Name `xml:"T:AnulaTicketBai"`
	TNamespace string   `xml:"xmlns:T,attr"`

	Cabecera   *Cabecera
	IDFactura  *IDFactura
	HuellaTBAI *HuellaTBAI
	Signature  *xmldsig.Signature `xml:"ds:Signature,omitempty"`

	zone l10n.Code // copied from invoice
}

// IDFactura contains the info to identify the invoice to cancel
type IDFactura struct {
	Emisor          *Emisor
	CabeceraFactura *CabeceraAnulacionFactura
}

// CabeceraAnulacionFactura contains the header of the invoice to cancel
type CabeceraAnulacionFactura struct {
	SerieFactura           string `xml:",omitempty"`
	NumFactura             string
	FechaExpedicionFactura string
}

// NewAnulaTicketBAI creates a new AnulaTicketBAI document
func NewAnulaTicketBAI(inv *bill.Invoice, ts time.Time) (*AnulaTicketBAI, error) {
	doc := &AnulaTicketBAI{
		TNamespace: ticketBAIAnulacionNamespace,
		Cabecera: &Cabecera{
			IDVersionTBAI: ticketBAIVersion,
		},
		IDFactura: &IDFactura{
			Emisor: newEmisor(inv.Supplier),
			CabeceraFactura: &CabeceraAnulacionFactura{
				SerieFactura:           inv.Series,
				NumFactura:             inv.Code,
				FechaExpedicionFactura: formatDate(ts),
			},
		},
	}

	doc.zone = inv.Supplier.TaxID.Zone

	return doc, nil
}

// Fingerprint calculates the fingerprint of the document
func (doc *AnulaTicketBAI) Fingerprint(conf *FingerprintConfig) error {
	doc.HuellaTBAI = newHuellaTBAI(conf)
	return nil
}

// Sign signs the document with the given certificate and role
func (doc *AnulaTicketBAI) Sign(docID string, cert *xmldsig.Certificate, role IssuerRole, opts ...xmldsig.Option) error {
	s, err := newSignature(doc, docID, doc.zone, role, cert, opts...)
	if err != nil {
		return err
	}

	doc.Signature = s

	return nil
}

// Bytes returns the XML document bytes
func (doc *AnulaTicketBAI) Bytes() ([]byte, error) {
	return toBytes(doc)
}

// BytesIndent returns the indented XML document bytes
func (doc *AnulaTicketBAI) BytesIndent() ([]byte, error) {
	return toBytes(doc)
}
