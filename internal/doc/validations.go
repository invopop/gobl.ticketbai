package doc

import (
	"errors"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
)

var validSupplierLocalities = []l10n.Code{
	ZoneBI, // Vizcaya
	ZoneSS, // Guizpuzcoa
	ZoneVI, // Álava
}

func validate(inv *bill.Invoice, zone l10n.Code) error {
	if inv.Type == bill.InvoiceTypeCorrective {
		return errors.New("corrective invoices not supported, use credit or debit notes")
	}

	if inv.Supplier == nil || inv.Supplier.TaxID == nil {
		return nil // ignore
	}

	if zone == l10n.CodeEmpty {
		return errors.New("zone is required")
	}

	if !zone.In(validSupplierLocalities...) {
		return errors.New("zone not supported by TicketBAI")
	}

	if zone.In(ZoneSS, ZoneVI) {
		if len(inv.Lines) > 1000 {
			return errors.New("line count over limit (1000) for tax locality")
		}
		if inv.Customer != nil && len(inv.Customer.Addresses) == 0 {
			return errors.New("customer address required")
		}
	}

	for _, l := range inv.Lines {
		if len(l.Charges) > 0 {
			return errors.New("charges are not supported")
		}
	}

	if len(inv.Charges) > 0 {
		return errors.New("charges are not supported")
	}

	return nil
}
