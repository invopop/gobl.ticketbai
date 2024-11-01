// Package gateways contains the different interfaces to send the TicketBAI
// documents to.
package gateways

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
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
	ErrConnection       = newError("connection")
	ErrInvalidRequest   = newError("invalid-request")
	ErrDuplicatedRecord = newError("duplicate")
)

// Error allows for structured responses from the gateway to be able to
// response codes and messages.
type Error struct {
	key     string
	code    string
	message string
	cause   error
}

// Error produces a human readable error message.
func (e *Error) Error() string {
	out := []string{e.key}
	if e.code != "" {
		out = append(out, e.code)
	}
	if e.message != "" {
		out = append(out, e.message)
	}
	return strings.Join(out, ": ")
}

// Message returns the human message for the error.
func (e *Error) Message() string {
	return e.message
}

// Code returns the code provided by the remote service.
func (e *Error) Code() string {
	return e.code
}

func newError(key string) *Error {
	return &Error{key: key}
}

// withCode duplicates and adds the code to the error.
func (e *Error) withCode(code string) *Error {
	e = e.clone()
	e.code = code
	return e
}

// withMessage duplicates and adds the message to the error.
func (e *Error) withMessage(msg string) *Error {
	e = e.clone()
	e.message = msg
	return e
}

func (e *Error) withCause(err error) *Error {
	e = e.clone()
	e.cause = err
	e.message = err.Error()
	return e
}

func (e *Error) clone() *Error {
	ne := new(Error)
	*ne = *e
	return ne
}

// Is checks to see if the target error is the same as the current one
// or forms part of the chain.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return errors.Is(e.cause, target)
	}
	return e.key == t.key
}

// Connection defines what is expected from a connection to a gateway.
type Connection interface {
	// Post sends the complete TicketBAI document to the remote end-point. We assume
	// the document has been fully prepared and signed.
	Post(ctx context.Context, inv *bill.Invoice, doc *doc.TicketBAI) error
	Fetch(ctx context.Context, nif string, name string, year int, page int, head *doc.CabeceraFactura) ([]*doc.TicketBAI, error)
	Cancel(ctx context.Context, inv *bill.Invoice, doc *doc.AnulaTicketBAI) error
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

	l.Register(doc.ZoneBI, newEbizkaia(env, tlsConf))
	l.Register(doc.ZoneSS, newGipuzkoa(env, tlsConf))

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
