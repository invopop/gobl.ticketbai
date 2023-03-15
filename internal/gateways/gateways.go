package gateways

import (
	"context"
	"fmt"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/xmldsig"
)

// Environment to use for connections
const (
	EnvProduction = "production"
	EnvTesting    = "testing"
)

// Error is used to provide more contextual errors
type Error string

// Standard gateway error responses
var (
	ErrRejected   Error = "rejected"
	ErrConnection Error = "connection"
)

// Error provides string form of an error
func (e Error) Error() string {
	return string(e)
}

// Connection defines what is expected from a connection to a gateway.
type Connection interface {
	// Post sends the complete TicketBAI document to the remote end-point. We assume
	// the document has been fully prepared and signed.
	Post(ctx context.Context, inv *bill.Invoice, tbai *doc.TicketBAI) error
}

// List keeps together the list of connections
type List struct {
	ebizkaia *EBizkaiaConn
}

// New instantiates a new set of connections using the provided config.
func New(env string, cert *xmldsig.Certificate) (*List, error) {
	l := new(List)
	tlsConf, err := cert.TLSAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("preparing TLS config: %w", err)
	}
	l.ebizkaia = newEbizkaia(env, tlsConf)
	return l, nil
}

// For provides the connection needed for the given locality, or nil,
// if not supported.
func (l *List) For(zone l10n.Code) Connection {
	switch zone {
	case es.ZoneBI:
		return l.ebizkaia
	}
	return nil
}
