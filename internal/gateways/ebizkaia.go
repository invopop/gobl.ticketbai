package gateways

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/go-resty/resty/v2"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/internal/gateways/ebizkaia"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/xmldsig"
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
	eBizkaiaN3ResponseInvalid = "Incorrecto"

	// Response codes of interest
	eBizkaiaN3RespCodeTechnical  = "B4_1000004" // “Error técnico”
	eBizkaiaN3RespCodeDuplicated = "B4_2000003" // “El registro no puede existir en el sistema”
	eBizkaiaN3RespCodeOther      = "N3_0000011" // “Otros, consulte el mensaje recibido (...)”
)

// Server-side error codes
var serverErrors = []string{
	eBizkaiaN3RespCodeTechnical,
	eBizkaiaN3RespCodeOther,
}

// EBizkaiaConn keeps all the connection details together for the Vizcaya region.
type EBizkaiaConn struct {
	client *resty.Client
}

func newEbizkaia(env Environment, tlsConfig *tls.Config) *EBizkaiaConn {
	c := new(EBizkaiaConn)
	c.client = resty.New()

	switch env {
	case EnvironmentProduction:
		c.client.SetBaseURL(eBizkaiaProductionBaseURL)
	default:
		c.client.SetBaseURL(eBizkaiaTestingBaseURL)
	}

	tlsConfig.InsecureSkipVerify = true
	c.client.SetTLSClientConfig(tlsConfig)
	c.client.SetDebug(os.Getenv("DEBUG") == "true")

	return c
}

// Post sends the complete TicketBAI document to the remote end-point. We assume
// the document has been signed and prepared.
func (c *EBizkaiaConn) Post(ctx context.Context, inv *bill.Invoice, doc *doc.TicketBAI) error {
	payload, err := doc.Bytes()
	if err != nil {
		return fmt.Errorf("generating payload: %w", err)
	}

	sup := &ebizkaia.Supplier{
		Year: inv.IssueDate.Year,
		NIF:  inv.Supplier.TaxID.Code.String(),
		Name: inv.Supplier.Name,
	}

	req, err := ebizkaia.NewCreateRequest(sup, payload)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp := ebizkaia.LROEPJ240FacturasEmitidasConSGAltaRespuesta{}

	err = c.sendRequest(ctx, req, eBizkaiaExecutePath, &resp)
	if errors.Is(err, ErrInvalidRequest) {
		if resp.FirstErrorCode() == eBizkaiaN3RespCodeDuplicated {
			return ErrDuplicatedRecord
		}

		if resp.FirstErrorDescription() != "" {
			return fmt.Errorf("ebizcaia: %w: %v", ErrInvalidRequest, resp.FirstErrorDescription())
		}
	}

	return err
}

// Fetch retrieves the TicketBAI from the remote end-point for the given
// taxpayer and year.
func (c *EBizkaiaConn) Fetch(ctx context.Context, nif string, name string, year int, page int, head *doc.CabeceraFactura) ([]*doc.TicketBAI, error) {
	sup := &ebizkaia.Supplier{
		Year: year,
		NIF:  nif,
		Name: name,
	}

	d, err := ebizkaia.NewFetchRequest(sup, page, head)
	if err != nil {
		return nil, fmt.Errorf("fetch request: %w", err)
	}

	resp := ebizkaia.LROEPJ240FacturasEmitidasConSGConsultaRespuesta{}
	if err := c.sendRequest(ctx, d, eBizkaiaQueryPath, &resp); err != nil {
		return nil, fmt.Errorf("sending fetch request: %w", err)
	}

	tbais := make([]*doc.TicketBAI, len(resp.FacturasEmitidas.FacturaEmitida))
	for i, f := range resp.FacturasEmitidas.FacturaEmitida {
		tbais[i] = buildTBAIDoc(f.TicketBai)
	}

	return tbais, nil
}

// Cancel sends the cancellation request for the TickeBAI invoice to the remote
// end-point.
func (c *EBizkaiaConn) Cancel(ctx context.Context, inv *bill.Invoice, doc *doc.AnulaTicketBAI) error {
	payload, err := doc.Bytes()
	if err != nil {
		return fmt.Errorf("generating payload: %w", err)
	}

	sup := &ebizkaia.Supplier{
		Year: inv.IssueDate.Year,
		NIF:  inv.Supplier.TaxID.Code.String(),
		Name: inv.Supplier.Name,
	}

	req, err := ebizkaia.NewCancelRequest(sup, payload)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.sendRequest(ctx, req, eBizkaiaExecutePath, nil)
}

func (c *EBizkaiaConn) sendRequest(ctx context.Context, doc *ebizkaia.Request, path string, resp interface{}) error {
	r := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("Accept-Encoding", "gzip").
		SetContentLength(true).
		SetHeaderVerbatim(eBizkaiaN3ContentTypeHeader, "application/xml").
		SetHeaderVerbatim(eBizkaiaN3DataHeader, string(doc.Header)).
		SetHeaderVerbatim(eBizkaiaN3VersionHeader, "1.0").
		SetBody(doc.Payload).
		SetResult(resp)

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
		if !slices.Contains(serverErrors, code) {
			// Not a server-side error, so the cause of it is in the request. We identify
			// it as an ErrInvalidRequest to handle it downstream.
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

// buildTBAIDoc builds a doc.TicketBAI from a TicketBAIType.
func buildTBAIDoc(f *ebizkaia.TicketBaiType) *doc.TicketBAI {
	return &doc.TicketBAI{
		Cabecera:   f.Cabecera,
		Sujetos:    f.Sujetos,
		Factura:    f.Factura,
		HuellaTBAI: f.HuellaTBAI,
		Signature: &xmldsig.Signature{
			Value: &xmldsig.Value{
				Value: f.Signature,
			},
		},
	}
}
