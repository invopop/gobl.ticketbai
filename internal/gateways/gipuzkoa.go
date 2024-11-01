package gateways

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl/bill"
)

const (
	// Requests
	gipuzkoaProductionBaseURL = "https://tbai-z.gipuzkoa.eus"
	gipuzkoaTestingBaseURL    = "https://tbai-z.prep.gipuzkoa.eus"

	gipuzkoaExecutePath = "/sarrerak/alta"
	gipuzkoaCancelPath  = "/sarrerak/baja"
)

const (
	GipuzkoaStatusReceived = "00" // looks good.
	GipuzkoaStatusRejected = "01" // there are errors that need fixing.
)

// GipuzkoaResponse defines the response fields from the Gipuzkoa region.
type GipuzkoaResponse struct {
	Output struct {
		ID                string `xml:"IdentificadoTBAI"`
		Data              string `xml:"FechaRecepcion"`
		Status            string `xml:"Estado"`
		Description       string `xml:"Descripcion"`
		BasqueDescription string `xml:"Azalpena"` // Description, but in Basque
		Errors            []struct {
			Code              string `xml:"Codigo"`
			Description       string `xml:"Descripcion"`
			BasqueDescription string `xml:"Azalpena"`
		} `xml:"ResultadosValidacion"`
		CSV string `xml:"CSV"` // Secure Verification Code
	} `xml:"Salida"`
}

// GipuzkoaConn keeps all the connection details together for the Gipuzkoa region.
type GipuzkoaConn struct {
	client *resty.Client
}

func newGipuzkoa(env Environment, tlsConfig *tls.Config) *GipuzkoaConn {
	c := new(GipuzkoaConn)
	c.client = resty.New()

	switch env {
	case EnvironmentProduction:
		c.client.SetBaseURL(gipuzkoaProductionBaseURL)
	default:
		c.client.SetBaseURL(gipuzkoaTestingBaseURL)
	}

	tlsConfig.InsecureSkipVerify = true
	tlsConfig.Renegotiation = tls.RenegotiateOnceAsClient
	c.client.SetTLSClientConfig(tlsConfig)
	c.client.SetDebug(os.Getenv("DEBUG") == "true")

	return c
}

// Post sends the complete TicketBAI document to the Gipuzkoa API.
func (c *GipuzkoaConn) Post(ctx context.Context, inv *bill.Invoice, doc *doc.TicketBAI) error {
	payload, err := doc.Bytes()
	if err != nil {
		return fmt.Errorf("generating payload: %w", err)
	}
	return c.post(ctx, gipuzkoaExecutePath, payload)
}

// Fetch will send a request to the Gipuzkoa API to fetch a previously issued document.
// Not currently supported.
func (c *GipuzkoaConn) Fetch(ctx context.Context, nif string, name string, year int, page int, head *doc.CabeceraFactura) ([]*doc.TicketBAI, error) {
	return nil, errors.New("not supported")
}

// Cancel will send a request to the Gipuzkoa API to cancel a previously issued document.
func (c *GipuzkoaConn) Cancel(ctx context.Context, inv *bill.Invoice, doc *doc.AnulaTicketBAI) error {
	payload, err := doc.Bytes()
	if err != nil {
		return fmt.Errorf("generating payload: %w", err)
	}
	return c.post(ctx, gipuzkoaCancelPath, payload)
}

func (c *GipuzkoaConn) post(ctx context.Context, path string, payload []byte) error {
	out := new(GipuzkoaResponse)
	req := c.client.R().
		SetContext(ctx).
		SetDebug(true).
		SetHeader("Content-Type", "application/xml").
		SetContentLength(true).
		SetBody(payload).
		SetResult(out)

	res, err := req.Post(path)
	if err != nil {
		return ErrConnection.withCause(err)
	}
	if res.StatusCode() != http.StatusOK {
		return ErrInvalidRequest.withCode(strconv.Itoa(res.StatusCode()))
	}

	if out.Output.Status != GipuzkoaStatusReceived {
		err := ErrInvalidRequest
		if len(out.Output.Errors) > 0 {
			e1 := out.Output.Errors[0]
			err = err.withMessage(e1.Description).withCode(e1.Code)
		}
		return err
	}

	return nil
}
