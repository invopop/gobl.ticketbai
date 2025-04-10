package doc

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/invopop/gobl/l10n"
	"github.com/sigurn/crc8"
)

// Codes contain info about the codes that should be generated and shown on a
// Ticketbai invoice. One is an alphanumeric code that identifies the invoice
// and the other one is a URL (which can be used by a customer to validate that
// the invoice has been sent to the tax agency) that should be encoded as a
// QR code in the printed invoice / ticket.
type Codes struct {
	TBAICode string
	QRCode   string
}

var crcTable = crc8.MakeTable(crc8.CRC8)

// generateCodes will generate the QR and TBAI codes for the invoice
func (doc *TicketBAI) generateCodes(locality l10n.Code) *Codes {
	tbaiCode := doc.generateTbaiCode()
	qrCode := doc.generateQRCode(locality, tbaiCode)

	return &Codes{
		TBAICode: tbaiCode,
		QRCode:   qrCode,
	}
}

func (doc *TicketBAI) generateTbaiCode() string {
	header := doc.Factura.CabeceraFactura
	dateParts := strings.Split(header.FechaExpedicionFactura, "-")
	date := dateParts[0] + dateParts[1] + dateParts[2][len(dateParts[2])-2:]

	signatureStart := fmt.Sprintf("%.13s", doc.SignatureValue())

	info := fmt.Sprintf("TBAI-%s-%s-%s-", doc.Sujetos.Emisor.NIF, date, signatureStart)

	crc := crc8.Checksum([]byte(info), crcTable)

	return fmt.Sprintf("%s%03d", info, crc)
}

func (doc *TicketBAI) generateQRCode(zone l10n.Code, tbaiCode string) string {
	var u string
	switch zone {
	case ZoneBI:
		u = "https://batuz.eus/QRTBAI/?"
	case ZoneSS:
		u = "https://tbai.egoitza.gipuzkoa.eus/qr/?"
	case ZoneVI:
		u = "https://ticketbai.araba.eus/tbai/qrtbai/?"
	default:
		return ""
	}

	query := []string{"id=" + url.QueryEscape(tbaiCode)}
	if doc.Factura.CabeceraFactura.SerieFactura != "" {
		query = append(query, "s="+url.QueryEscape(doc.Factura.CabeceraFactura.SerieFactura))
	}
	query = append(query,
		"nf="+url.QueryEscape(doc.Factura.CabeceraFactura.NumFactura),
		"i="+url.QueryEscape(doc.Factura.DatosFactura.ImporteTotalFactura),
	)
	u = u + strings.Join(query, "&")

	// Calculate the checksum
	cs := crc8.Checksum([]byte(u), crcTable)
	return fmt.Sprintf("%s&cr=%03d", u, cs)
}
