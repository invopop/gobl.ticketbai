// Package gateways contains the different interfaces to send the TicketBAI
// documents to.
package gateways

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/invopop/gobl.ticketbai/doc"
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
	ErrConnection = newError("connection")
	ErrInvalid    = newError("invalid")
	ErrDuplicate  = newError("duplicate")
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

// Key returns the key for the error.
func (e *Error) Key() string {
	return e.key
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
	Post(ctx context.Context, doc *doc.TicketBAI) error
	Cancel(ctx context.Context, doc *doc.AnulaTicketBAI) error
}

// New instantiates a new connection for the given zone and environment.
func New(env Environment, zone l10n.Code, cert *xmldsig.Certificate) (Connection, error) {
	tlsConf, err := cert.TLSAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("preparing TLS config: %w", err)
	}

	switch zone {
	case doc.ZoneBI:
		return newEbizkaia(env, tlsConf), nil
	case doc.ZoneSS:
		return newGipuzkoa(env, tlsConf), nil
	case doc.ZoneVI:
		return newAraba(env, tlsConf), nil
	}

	return nil, fmt.Errorf("zone %s not supported", zone)
}
