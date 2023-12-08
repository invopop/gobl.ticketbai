package doc

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
)

var validSupplierLocalities = []l10n.Code{
	es.ZoneBI, // Vizcaya
	es.ZoneSS, // Guizpuzcoa
	es.ZoneVI, // Álava
}

func validateInvoice(inv *bill.Invoice) error {
	if inv.Supplier == nil || inv.Supplier.TaxID == nil {
		return nil // ignore
	}

	tID := inv.Supplier.TaxID
	if tID.Zone == l10n.CodeEmpty {
		return errors.New("supplier tax identity locality is required")
	}

	if !tID.Zone.In(validSupplierLocalities...) {
		return errors.New("supplier tax identity locality not supported by TicketBAI")
	}

	if tID.Zone.In(es.ZoneSS, es.ZoneVI) {
		if len(inv.Lines) > 1000 {
			return errors.New("line count over limit (1000) for tax locality")
		}
		if inv.Customer != nil && len(inv.Customer.Addresses) == 0 {
			return errors.New("customer address required")
		}
	}

	return nil
}
