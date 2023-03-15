package gateways

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/internal/gateways/ebizkaia"
	"github.com/invopop/gobl/bill"
)

const (
	// Requests
	eBizkaiaProductionBaseURL   = "https://sarrerak.bizkaia.eus"
	eBizkaiaTestingBaseURL      = "https://pruesarrerak.bizkaia.eus"
	eBizkaiaExecutePath         = "/N3B4000M/aurkezpena"
	eBizkaiaN3VersionHeader     = "eus-bizkaia-n3-version"
	eBizkaiaN3ContentTypeHeader = "eus-bizkaia-n3-content-type"
	eBizkaiaN3DataHeader        = "eus-bizkaia-n3-data"

	// Responses
	eBizkaiaN3MessageHeader   = "Eus-Bizkaia-N3-Mensaje-Respuesta"
	eBizkaiaN3ResponseHeader  = "Eus-Bizkaia-N3-Tipo-Respuesta"
	eBizkaiaN3RegNumberHeader = "Eus-Bizkaia-N3-Numero-Registro"

	eBizkaiaN3ResponseInvalid = "Incorrecto"
)

// EBizkaiaConn keeps all the connection details together for the Vizcaya region.
type EBizkaiaConn struct {
	client *resty.Client
}

func newEbizkaia(env string, tlsConfig *tls.Config) *EBizkaiaConn {
	c := new(EBizkaiaConn)
	c.client = resty.New()
	switch env {
	case EnvProduction:
		c.client = c.client.SetBaseURL(eBizkaiaProductionBaseURL)
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
func (c *EBizkaiaConn) Post(ctx context.Context, inv *bill.Invoice, tbai *doc.TicketBAI) error {
	sup := &ebizkaia.Supplier{
		Year: inv.IssueDate.Year,
		NIF:  inv.Supplier.TaxID.Code.String(),
		Name: inv.Supplier.Name,
	}
	doc, err := ebizkaia.NewCreateRequest(sup, tbai)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	r := c.client.R().
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("Accept-Encoding", "gzip").
		SetContentLength(true).
		SetHeaderVerbatim(eBizkaiaN3ContentTypeHeader, "application/xml").
		SetHeaderVerbatim(eBizkaiaN3DataHeader, string(doc.Header)).
		SetHeaderVerbatim(eBizkaiaN3VersionHeader, "1.0").
		SetBody(doc.Payload)

	res, err := r.Post(eBizkaiaExecutePath)
	if err != nil {
		return fmt.Errorf("%w: ebizkaia: %s", ErrConnection, err.Error())
	}
	if res.StatusCode() != 200 {
		return fmt.Errorf("ebizkaia: status %d", res.StatusCode())
	}

	code := res.Header().Get(eBizkaiaN3ResponseHeader)
	if code == eBizkaiaN3ResponseInvalid {
		msg := res.Header().Get(eBizkaiaN3MessageHeader)
		return fmt.Errorf("ebizkaia: %v", msg)
	}

	return nil
}
