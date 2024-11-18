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
	var pat string
	switch zone {
	case ZoneBI:
		pat = "https://batuz.eus/QRTBAI/?id=%s&s=%s&nf=%s&i=%s"
	case ZoneSS:
		pat = "https://tbai.egoitza.gipuzkoa.eus/qr/?id=%s&s=%s&nf=%s&i=%s"
	case ZoneVI:
		pat = "https://ticketbai.araba.eus/tbai/qrtbai/?id=%s&s=%s&nf=%s&i=%s"
	default:
		return ""
	}

	tbaiCode = url.QueryEscape(tbaiCode)
	invCode := url.QueryEscape(doc.Factura.CabeceraFactura.NumFactura)
	invSeries := url.QueryEscape(doc.Factura.CabeceraFactura.SerieFactura)
	invTotal := doc.Factura.DatosFactura.ImporteTotalFactura

	qrCodeInfo := fmt.Sprintf(pat, tbaiCode, invSeries, invCode, invTotal)
	qrCodeCRC := crc8.Checksum([]byte(qrCodeInfo), crcTable)

	return fmt.Sprintf("%s&cr=%03d", qrCodeInfo, qrCodeCRC)
}
