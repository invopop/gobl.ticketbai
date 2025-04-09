package doc_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl.ticketbai/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacturaConversion(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2022-08-15T22:15:05+02:00")
	require.NoError(t, err)
	role := doc.IssuerRoleThirdParty

	t.Run("should add info about id of an invoice", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Code = "something-001"
		goblInvoice.Series = "SERIES"

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "something-001", factura.CabeceraFactura.NumFactura)
		assert.Equal(t, "SERIES", factura.CabeceraFactura.SerieFactura)
	})

	t.Run("should add issue time / date info", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "15-08-2022", factura.CabeceraFactura.FechaExpedicionFactura)
		assert.Equal(t, "22:15:05", factura.CabeceraFactura.HoraExpedicionFactura) // Europe/Madrid time
	})

	t.Run("should mark an invoice as simplified (ticket)", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.SetTags(tax.TagSimplified)

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "S", factura.CabeceraFactura.FacturaSimplificada)
	})

	t.Run("should fill invoice operation date", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.OperationDate = cal.NewDate(2022, 3, 15)

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "15-03-2022", factura.DatosFactura.FechaOperacion)
	})

	t.Run("should fill invoice description from general note", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Notes = []*org.Note{
			{Key: org.NoteKeyGeneral, Text: "Description of invoice"},
		}

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "Description of invoice", factura.DatosFactura.DescripcionFactura)
	})

	t.Run("should return error if no description (general note) found", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Notes = []*org.Note{}

		_, err := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		assert.ErrorContains(t, err, "notes: missing note with key 'general'")
	})

	t.Run("should include VAT and discounts to the total of the invoice", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:     1,
			Quantity:  num.MakeAmount(100, 0),
			Item:      &org.Item{Name: "A", Price: num.NewAmount(11, 0)},
			Discounts: []*bill.LineDiscount{DiscountOf(100)},
			Taxes:     tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "1210.00", factura.DatosFactura.ImporteTotalFactura)
	})

	t.Run("should not include retained taxes (IRPF) to the total of the invoice", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.NewAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{Category: "VAT", Rate: "standard"},
				&tax.Combo{Category: "IRPF", Rate: "pro"},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "1210.00", factura.DatosFactura.ImporteTotalFactura)
	})

	t.Run("should add retained taxes (IRPF) to RetencionSoportada", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.NewAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{Category: "VAT", Rate: "standard"},
				&tax.Combo{Category: "IRPF", Rate: "pro"},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		factura := invoice.Factura
		assert.Equal(t, "150.00", factura.DatosFactura.RetencionSoportada)
	})

	t.Run("should add general regime (01) if no other VAT keys", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.NewAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{Category: "VAT", Rate: "standard"},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		claves := invoice.Factura.DatosFactura.Claves
		assert.Equal(t, "01", claves.IDClave[0].ClaveRegimenIvaOpTrascendencia)
	})

	t.Run("should add export (02) if the client is foreign", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID.Country = "GB"

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		claves := invoice.Factura.DatosFactura.Claves
		assert.Equal(t, "02", claves.IDClave[0].ClaveRegimenIvaOpTrascendencia)
	})

	t.Run("should add surcharge key (51) if any line is surcharged", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item: &org.Item{
				Name:  "A",
				Price: num.NewAmount(10, 0),
			},
			Taxes: tax.Set{
				&tax.Combo{
					Category: "VAT",
					Rate:     "standard",
					Ext:      tax.Extensions{tbai.ExtKeyProduct: "resale"},
				},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		claves := invoice.Factura.DatosFactura.Claves
		assert.Equal(t, "51", claves.IDClave[0].ClaveRegimenIvaOpTrascendencia)
	})

	t.Run("should add simplified tax regime (52) is the issuer works this way",
		func(t *testing.T) {
			goblInvoice := test.LoadInvoice("sample-invoice.json")
			goblInvoice.SetTags(es.TagSimplifiedScheme)

			invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

			claves := invoice.Factura.DatosFactura.Claves
			assert.Equal(t, "52", claves.IDClave[0].ClaveRegimenIvaOpTrascendencia)
		})
}

func DiscountOf(amount int) *bill.LineDiscount {
	return &bill.LineDiscount{
		Amount: num.MakeAmount(int64(amount), 0),
		Reason: "No reason",
	}
}
