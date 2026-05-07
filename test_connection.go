package ticketbai

import (
	"context"

	"github.com/invopop/gobl.ticketbai/convert"
	"github.com/invopop/gobl.ticketbai/internal/gateways"
)

// TestConnection is a mock gateway connection for testing purposes
type TestConnection struct {
	postCalled   bool
	cancelCalled bool
}

var _ gateways.Connection = (*TestConnection)(nil)

// Post mocks the Post method of the Connection interface
func (tc *TestConnection) Post(_ context.Context, _ *convert.TicketBAI) error {
	tc.postCalled = true
	return nil
}

// Cancel mocks the Cancel method of the Connection interface
func (tc *TestConnection) Cancel(_ context.Context, _ *convert.AnulaTicketBAI) error {
	tc.cancelCalled = true
	return nil
}
