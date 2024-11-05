package ticketbai

import (
	"context"

	"github.com/invopop/gobl.ticketbai/doc"
	"github.com/invopop/gobl/bill"
)

// TestConnection is a mock gateway connection for testing purposes
type TestConnection struct {
	postCalled   bool
	fetchCalled  bool
	cancelCalled bool
}

// Post mocks the Post method of the Connection interface
func (tc *TestConnection) Post(_ context.Context, _ *bill.Invoice, _ *doc.TicketBAI) error {
	tc.postCalled = true
	return nil
}

// Cancel mocks the Cancel method of the Connection interface
func (tc *TestConnection) Cancel(_ context.Context, _ *bill.Invoice, _ *doc.AnulaTicketBAI) error {
	tc.cancelCalled = true
	return nil
}

// Fetch mocks the Fetch method of the Connection interface
func (tc *TestConnection) Fetch(_ context.Context, _ string, _ string, _ int, _ int, _ *doc.CabeceraFactura) ([]*doc.TicketBAI, error) {
	tc.fetchCalled = true
	return nil, nil
}
