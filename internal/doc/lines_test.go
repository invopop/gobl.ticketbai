package doc_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLines(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	require.NoError(t, err)
	role := doc.IssuerRoleThirdParty

	t.Run("should show line info", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes:    tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		lines := invoice.Factura.DatosFactura.DetallesFactura.IDDetalleFactura
		assert.Equal(t, 1, len(lines))
		assert.Equal(t, "A", lines[0].DescripcionDetalle)
		assert.Equal(t, "100", lines[0].Cantidad)
		assert.Equal(t, "10.00", lines[0].ImporteUnitario)
		assert.Equal(t, "1210.00", lines[0].ImporteTotal)
	})

	t.Run("should show line discount", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:     1,
			Quantity:  num.MakeAmount(100, 0),
			Item:      &org.Item{Name: "A", Price: num.MakeAmount(11, 0)},
			Discounts: []*bill.LineDiscount{DiscountOf(100)},
			Taxes:     tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		line := invoice.Factura.DatosFactura.DetallesFactura.IDDetalleFactura[0]
		assert.Equal(t, "100.00", line.Descuento)
		assert.Equal(t, "1210.00", line.ImporteTotal)
	})

	t.Run("should subtract taxes if included in prices per unit", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Tax = &bill.Tax{PricesInclude: "VAT"}

		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(10, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(121, 0)},
			Taxes:    tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		line := invoice.Factura.DatosFactura.DetallesFactura.IDDetalleFactura[0]
		assert.Equal(t, "100.00", line.ImporteUnitario)
		assert.Equal(t, "1210.00", line.ImporteTotal)
	})

	t.Run("should return error if more than 1000 lines included and not Vizcaya", func(t *testing.T) {
		inv, _ := test.LoadInvoice("sample-invoice.json")
		inv.Lines = []*bill.Line{}
		for i := 1; i <= 1001; i++ {
			inv.Lines = append(inv.Lines, &bill.Line{
				Index:    1,
				Quantity: num.MakeAmount(100, 0),
				Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
				Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Rate: tax.RateStandard}},
			})
		}
		require.NoError(t, inv.Calculate())

		_, err := doc.NewTicketBAI(inv, ts, role, doc.ZoneSS)

		assert.ErrorContains(t, err, "line count over limit (1000) for tax locality")
	})
}
