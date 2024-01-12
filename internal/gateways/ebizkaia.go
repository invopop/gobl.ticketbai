package gateways

import (
	"crypto/tls"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/invopop/gobl.ticketbai/internal/gateways/ebizkaia"
	"github.com/invopop/gobl/bill"
	"golang.org/x/net/html/charset"
)

const (
	// Requests
	eBizkaiaProductionBaseURL   = "https://sarrerak.bizkaia.eus"
	eBizkaiaTestingBaseURL      = "https://pruesarrerak.bizkaia.eus"
	eBizkaiaExecutePath         = "/N3B4000M/aurkezpena"
	eBizkaiaQueryPath           = "/N3B4001M/kontsulta"
	eBizkaiaN3VersionHeader     = "eus-bizkaia-n3-version"
	eBizkaiaN3ContentTypeHeader = "eus-bizkaia-n3-content-type"
	eBizkaiaN3DataHeader        = "eus-bizkaia-n3-data"

	// Responses
	eBizkaiaN3MessageHeader   = "Eus-Bizkaia-N3-Mensaje-Respuesta"
	eBizkaiaN3ResponseHeader  = "Eus-Bizkaia-N3-Tipo-Respuesta"
	eBizkaiaN3RespCodeHeader  = "Eus-Bizkaia-N3-Codigo-Respuesta"
	eBizkaiaN3RegNumberHeader = "Eus-Bizkaia-N3-Numero-Registro"

	eBizkaiaN3ResponseInvalid   = "Incorrecto"
	eBizkaiaN3RespCodeTechnical = "B4_1000004"
)

// EBizkaiaConn keeps all the connection details together for the Vizcaya region.
type EBizkaiaConn struct {
	client *resty.Client
}

func newEbizkaia(env Environment, tlsConfig *tls.Config) *EBizkaiaConn {
	c := new(EBizkaiaConn)
	c.client = resty.New()
	switch env {
	case EnvironmentProduction:
		c.client = c.client.SetBaseURL(eBizkaiaProductionBaseURL)
		c.client.SetDebug(true)
		tlsConfig.InsecureSkipVerify = true
	default:
		c.client = c.client.SetBaseURL(eBizkaiaTestingBaseURL)
		c.client.SetDebug(true)
		tlsConfig.InsecureSkipVerify = true
	}

	c.client.SetTLSClientConfig(tlsConfig)

	return c
}

// Post sends the complete TicketBAI document to the remote end-point. We assume
// the document has been signed and prepared.
func (c *EBizkaiaConn) Post(inv *bill.Invoice, payload []byte) error {
	sup := &ebizkaia.Supplier{
		Year: inv.IssueDate.Year,
		NIF:  inv.Supplier.TaxID.Code.String(),
		Name: inv.Supplier.Name,
	}

	doc, err := ebizkaia.NewCreateRequest(sup, payload)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.sendRequest(doc, eBizkaiaExecutePath)
}

// Fetch retrieves the TicketBAI from the remote end-point for the given
// taxpayer and year.
func (c *EBizkaiaConn) Fetch(nif string, name string, year int) error {
	sup := &ebizkaia.Supplier{
		Year: year,
		NIF:  nif,
		Name: name,
	}

	doc, err := ebizkaia.NewFetchRequest(sup)
	if err != nil {
		return fmt.Errorf("fetch request: %w", err)
	}

	return c.sendRequest(doc, eBizkaiaQueryPath)
}

// Cancel sends the cancellation request for the TickeBAI invoice to the remote
// end-point.
func (c *EBizkaiaConn) Cancel(inv *bill.Invoice, payload []byte) error {
	sup := &ebizkaia.Supplier{
		Year: inv.IssueDate.Year,
		NIF:  inv.Supplier.TaxID.Code.String(),
		Name: inv.Supplier.Name,
	}

	doc, err := ebizkaia.NewCancelRequest(sup, payload)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.sendRequest(doc, eBizkaiaExecutePath)
}

func (c *EBizkaiaConn) sendRequest(doc *ebizkaia.Request, path string) error {
	r := c.client.R().
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("Accept-Encoding", "gzip").
		SetContentLength(true).
		SetHeaderVerbatim(eBizkaiaN3ContentTypeHeader, "application/xml").
		SetHeaderVerbatim(eBizkaiaN3DataHeader, string(doc.Header)).
		SetHeaderVerbatim(eBizkaiaN3VersionHeader, "1.0").
		SetBody(doc.Payload)

	res, err := r.Post(path)
	if err != nil {
		return fmt.Errorf("%w: ebizkaia: %s", ErrConnection, err.Error())
	}
	if res.StatusCode() != 200 {
		return fmt.Errorf("ebizkaia: status %d", res.StatusCode())
	}

	code := res.Header().Get(eBizkaiaN3ResponseHeader)
	if code == eBizkaiaN3ResponseInvalid {
		msg := res.Header().Get(eBizkaiaN3MessageHeader)
		msg = convertToUTF8(msg)

		code := res.Header().Get(eBizkaiaN3RespCodeHeader)
		if code != eBizkaiaN3RespCodeTechnical {
			// Not a technical error, so the cause of it is in the request. We identify
			// it as an ErrInvalidRequest to handle it properly later.
			return fmt.Errorf("ebizcaia: %w: %v: %v", ErrInvalidRequest, code, msg)
		}

		return fmt.Errorf("ebizkaia: %v: %v", code, msg)
	}

	return nil
}

// convertToValidUTF8 determines the encoding of a string and converts it to
// UTF-8. Certain strings returned by eBizkaia aren't in UTF-8.
func convertToUTF8(s string) string {
	e, _, _ := charset.DetermineEncoding([]byte(s), "")
	out, _ := e.NewDecoder().Bytes([]byte(s))
	return string(out)
}
