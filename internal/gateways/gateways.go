// Package gateways contains the different interfaces to send the TicketBAI
// documents to.
package gateways

import (
	"errors"
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/xmldsig"
)

// Environment defines the environment to use for connections
type Environment string

// Environment to use for connections
const (
	EnvironmentProduction Environment = "production"
	EnvironmentTesting    Environment = "testing"
)

// Standard gateway error responses
var (
	ErrConnection     = errors.New("connection")
	ErrInvalidRequest = errors.New("invalid request")
)

// Connection defines what is expected from a connection to a gateway.
type Connection interface {
	// Post sends the complete TicketBAI document to the remote end-point. We assume
	// the document has been fully prepared and signed.
	Post(inv *bill.Invoice, payload []byte) error
	Fetch(nif string, name string, year int) error
	Cancel(inv *bill.Invoice, payload []byte) error
}

// List keeps together the list of connections
type List struct {
	conns map[l10n.Code]Connection
}

// New instantiates a new set of connections using the provided config.
func New(env Environment, cert *xmldsig.Certificate) (*List, error) {
	l := new(List)

	tlsConf, err := cert.TLSAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("preparing TLS config: %w", err)
	}

	l.Register(es.ZoneBI, newEbizkaia(env, tlsConf))

	return l, nil
}

// Register adds a connection to use in a zone
func (l *List) Register(zone l10n.Code, c Connection) {
	if l.conns == nil {
		l.conns = make(map[l10n.Code]Connection)
	}
	l.conns[zone] = c
}

// For provides the connection needed for the given zone, or nil, if not
// supported.
func (l *List) For(zone l10n.Code) Connection {
	if l.conns == nil {
		return nil
	}
	return l.conns[zone]
}
