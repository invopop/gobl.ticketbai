package doc_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesgloseConversion(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	require.NoError(t, err)
	role := doc.IssuerRoleThirdParty

	t.Run("should fill DesgloseFactura when customer is from Spain", func(t *testing.T) {
		goblInvoice := invoiceFromCountry(l10n.ES.Tax())

		invoice, err := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)
		require.NoError(t, err)

		factura := invoice.Factura
		assert.NotNil(t, factura.TipoDesglose.DesgloseFactura)
	})

	t.Run("should fill DesgloseFactura when there is no customer (simplified invoice / ticket)",
		func(t *testing.T) {
			goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
			goblInvoice.Customer = nil

			invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

			desglose := invoice.Factura.TipoDesglose
			assert.NotNil(t, desglose.DesgloseFactura)
		})

	t.Run("should fill DesgloseTipoOperacion when customer is from other country",
		func(t *testing.T) {
			goblInvoice := invoiceFromCountry("GB")

			invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

			desglose := invoice.Factura.TipoDesglose
			assert.NotNil(t, desglose.DesgloseTipoOperacion)
		})

	t.Run("should distinguish goods from services when customer from other country",
		func(t *testing.T) {
			goblInvoice := invoiceFromCountry("GB")
			goblInvoice.Lines[0].Taxes.Get(tax.CategoryVAT).Ext = tax.Extensions{tbai.ExtKeyProduct: "goods"}
			_ = goblInvoice.Calculate()

			invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

			details := invoice.Factura.TipoDesglose.DesgloseTipoOperacion
			assert.NotNil(t, details.Entrega)
			assert.Nil(t, details.PrestacionServicios)
		})

	t.Run("should use services instead of goods as default when customer from other country",
		func(t *testing.T) {
			goblInvoice := invoiceFromCountry("GB")
			goblInvoice.Lines[0].Item.Key = cbc.KeyEmpty

			invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

			details := invoice.Factura.TipoDesglose.DesgloseTipoOperacion
			assert.NotNil(t, details.PrestacionServicios)
			assert.Nil(t, details.Entrega)
		})

	t.Run("should divide details between services and goods", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("GB")
		goblInvoice.SetTags(tax.TagCustomerRates)
		goblInvoice.Lines = []*bill.Line{
			{
				Index:    1,
				Quantity: num.MakeAmount(1, 0),
				Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
				Taxes: tax.Set{
					&tax.Combo{
						Category: "VAT",
						Rate:     "standard",
					},
				},
			},
			{
				Index:    2,
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "A",
					Price: num.MakeAmount(20, 0),
				},
				Taxes: tax.Set{
					&tax.Combo{
						Category: "VAT",
						Rate:     "standard",
						Ext: tax.Extensions{
							tbai.ExtKeyProduct: "goods",
						},
					},
				},
			},
			{
				Index:    3,
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "A",
					Price: num.MakeAmount(10, 0),
				},
				Taxes: tax.Set{
					&tax.Combo{
						Category: "VAT",
						Rate:     "reduced",
						Ext: tax.Extensions{
							tbai.ExtKeyProduct: "goods",
						},
					},
				},
			},
		}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		details := invoice.Factura.TipoDesglose.DesgloseTipoOperacion
		if assert.NotNil(t, details.Entrega.NoSujeta) {
			assert.Equal(t, 1, len(details.Entrega.NoSujeta.DetalleNoSujeta))
			assert.Equal(t, "30.00", details.Entrega.NoSujeta.DetalleNoSujeta[0].Importe.String())
		}
		if assert.NotNil(t, details.PrestacionServicios.NoSujeta) {
			assert.Equal(t, 1, len(details.PrestacionServicios.NoSujeta.DetalleNoSujeta))
			assert.Equal(t, "10.00", details.PrestacionServicios.NoSujeta.DetalleNoSujeta[0].Importe.String())
		}
	})

	t.Run("should divide details between services and goods when taxes exist",
		func(t *testing.T) {
			goblInvoice := invoiceFromCountry("GB")
			goblInvoice.Lines = []*bill.Line{
				{
					Index:    1,
					Quantity: num.MakeAmount(1, 0),
					Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
					Taxes:    tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
				},
				{
					Index:    2,
					Quantity: num.MakeAmount(1, 0),
					Item: &org.Item{
						Name:  "A",
						Price: num.MakeAmount(20, 0),
					},
					Taxes: tax.Set{
						&tax.Combo{
							Category: "VAT",
							Rate:     "standard",
							Ext:      tax.Extensions{tbai.ExtKeyProduct: "goods"},
						},
					},
				},
			}
			_ = goblInvoice.Calculate()

			invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

			details := invoice.Factura.TipoDesglose.DesgloseTipoOperacion
			goodsDetails := details.Entrega.Sujeta.NoExenta.DetalleNoExenta[0]
			serviceDetails := details.PrestacionServicios.Sujeta.NoExenta.DetalleNoExenta[0]
			assert.Equal(t, "20.00", goodsDetails.DesgloseIVA.DetalleIVA[0].BaseImponible)
			assert.Equal(t, "10.00", serviceDetails.DesgloseIVA.DetalleIVA[0].BaseImponible)
		})

	t.Run("should add No Sujeta when there is no VAT tax even with IRPF", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{{
			Index:     1,
			Quantity:  num.MakeAmount(100, 0),
			Item:      &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Discounts: []*bill.LineDiscount{DiscountOf(100)},
			Taxes: tax.Set{
				&tax.Combo{Category: "IRPF", Rate: "pro"},
				&tax.Combo{
					Category: "VAT",
					Rate:     "exempt",
					Ext:      tax.Extensions{tbai.ExtKeyExemption: "OT"},
				},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		assert.Equal(t, "900.00", desglose.NoSujeta.DetalleNoSujeta[0].Importe.String())
		assert.Equal(t, "OT", desglose.NoSujeta.DetalleNoSujeta[0].Causa)
	})

	t.Run("should change No Sujeta cause when taxes are paid in other EU country", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{
					Category: "VAT",
					Rate:     "exempt",
					Ext:      tax.Extensions{tbai.ExtKeyExemption: "RL"},
				},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		assert.Equal(t, "RL", desglose.NoSujeta.DetalleNoSujeta[0].Causa)
	})

	t.Run("should add No Sujeta where there is no VAT on foreign invoices", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("GB")
		goblInvoice.Lines = []*bill.Line{{
			Index:     1,
			Quantity:  num.MakeAmount(100, 0),
			Item:      &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Discounts: []*bill.LineDiscount{DiscountOf(100)},
			Taxes: tax.Set{
				&tax.Combo{
					Category: "VAT",
					Rate:     "exempt",
					Ext:      tax.Extensions{tbai.ExtKeyExemption: "RL"},
				},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseTipoOperacion
		assert.Equal(t, "900.00", desglose.PrestacionServicios.NoSujeta.DetalleNoSujeta[0].Importe.String())
	})

	t.Run("should add VAT detail on national invoices", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{Category: "VAT", Rate: "standard"},
				&tax.Combo{Category: "VAT", Rate: "reduced"},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		desgloseIVA := desglose.Sujeta.NoExenta.DetalleNoExenta[0].DesgloseIVA
		detail := detalleIVA(desgloseIVA, "21.00")
		assert.Equal(t, "1000.00", detail.BaseImponible)
		assert.Equal(t, "21.00", detail.TipoImpositivo)
		assert.Equal(t, "210.00", detail.CuotaImpuesto)
		detailReduced := detalleIVA(desgloseIVA, "10.00")
		assert.Equal(t, "1000.00", detailReduced.BaseImponible)
		assert.Equal(t, "10.00", detailReduced.TipoImpositivo)
		assert.Equal(t, "100.00", detailReduced.CuotaImpuesto)
	})

	t.Run("should add VAT detail on foreign invoices", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("GB")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Rate: tax.RateStandard}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseTipoOperacion
		desgloseIVA := desglose.PrestacionServicios.Sujeta.NoExenta.DetalleNoExenta[0].DesgloseIVA
		assert.Equal(t, "1000.00", desgloseIVA.DetalleIVA[0].BaseImponible)
		assert.Equal(t, "21.00", desgloseIVA.DetalleIVA[0].TipoImpositivo)
		assert.Equal(t, "210.00", desgloseIVA.DetalleIVA[0].CuotaImpuesto)
	})

	t.Run("should add equivalence surcharge info", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{Category: "VAT", Rate: "standard+eqs"},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		desgloseIVA := desglose.Sujeta.NoExenta.DetalleNoExenta[0].DesgloseIVA
		assert.Equal(t, "5.20", desgloseIVA.DetalleIVA[0].TipoRecargoEquivalencia)
		assert.Equal(t, "52.00", desgloseIVA.DetalleIVA[0].CuotaRecargoEquivalencia)
	})

	t.Run("should mark lines that are under eq. surcharge (provider paid extra taxes)", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{
			{
				Index:    1,
				Quantity: num.MakeAmount(1, 0),
				Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
				Taxes:    tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
			},
			{
				Index:    2,
				Quantity: num.MakeAmount(100, 0),
				Item: &org.Item{
					Name:  "A",
					Price: num.MakeAmount(10, 0),
				},
				Taxes: tax.Set{
					&tax.Combo{
						Category: "VAT",
						Rate:     "standard",
						Ext:      tax.Extensions{tbai.ExtKeyProduct: "resale"},
					},
				},
			},
		}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		desgloseIVA := desglose.Sujeta.NoExenta.DetalleNoExenta[0].DesgloseIVA
		assert.Equal(t, "", desgloseIVA.DetalleIVA[0].OperacionEnRecargoDeEquivalenciaORegimenSimplificado)
		assert.Equal(t, "10.00", desgloseIVA.DetalleIVA[0].BaseImponible)
		assert.Equal(t, "S", desgloseIVA.DetalleIVA[1].OperacionEnRecargoDeEquivalenciaORegimenSimplificado)
		assert.Equal(t, "1000.00", desgloseIVA.DetalleIVA[1].BaseImponible)
	})

	t.Run("should add VAT details when the rate is 0% on national invoices", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes: tax.Set{
				&tax.Combo{
					Category: tax.CategoryVAT,
					Rate:     tax.RateExempt,
					Ext:      tax.Extensions{tbai.ExtKeyExemption: "E1"},
				},
			},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		assert.Equal(t, "1000.00", desglose.Sujeta.Exenta.DetalleExenta[0].BaseImponible)
	})

	t.Run("should divide into multiple VAT details when 0% and multiple causes", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.Lines = []*bill.Line{
			{
				Index:    1,
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "A",
					Price: num.MakeAmount(10, 0),
				},
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						Rate:     tax.RateExempt,
						Ext: tax.Extensions{
							tbai.ExtKeyExemption: "E1",
						},
					},
				},
			},
			{
				Index:    2,
				Quantity: num.MakeAmount(1, 0),
				Item: &org.Item{
					Name:  "A",
					Price: num.MakeAmount(20, 0),
				},
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						Rate:     tax.RateExempt,
						Ext: tax.Extensions{
							tbai.ExtKeyExemption: "E2",
						},
					},
				},
			},
		}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		e1 := findExemption(desglose.Sujeta.Exenta.DetalleExenta, "E1")
		assert.NotNil(t, e1)
		assert.Equal(t, "10.00", e1.BaseImponible)
		e2 := findExemption(desglose.Sujeta.Exenta.DetalleExenta, "E2")
		assert.NotNil(t, e2)
		assert.Equal(t, "20.00", e2.BaseImponible)
	})

	t.Run("should mark lines if company works by modules (simplified regime)", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.SetTags(es.TagSimplifiedScheme)
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes:    tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		desgloseIVA := desglose.Sujeta.NoExenta.DetalleNoExenta[0].DesgloseIVA
		assert.Equal(t, "S", desgloseIVA.DetalleIVA[0].OperacionEnRecargoDeEquivalenciaORegimenSimplificado)
	})

	t.Run("should mark tax details if there is reverse charge", func(t *testing.T) {
		goblInvoice := invoiceFromCountry("ES")
		goblInvoice.SetTags(tax.TagReverseCharge)
		goblInvoice.Lines = []*bill.Line{{
			Index:    1,
			Quantity: num.MakeAmount(100, 0),
			Item:     &org.Item{Name: "A", Price: num.MakeAmount(10, 0)},
			Taxes:    tax.Set{&tax.Combo{Category: "VAT", Rate: "standard"}},
		}}
		_ = goblInvoice.Calculate()

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		desglose := invoice.Factura.TipoDesglose.DesgloseFactura
		assert.Equal(t, "S2", desglose.Sujeta.NoExenta.DetalleNoExenta[0].TipoNoExenta)
	})
}

func invoiceFromCountry(countryCode l10n.TaxCountryCode) *bill.Invoice {
	goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
	goblInvoice.Customer.TaxID.Country = countryCode
	return goblInvoice
}

func detalleIVA(desgloseIVA *doc.DesgloseIVA, rate string) *doc.DetalleIVA {
	for _, detail := range desgloseIVA.DetalleIVA {
		if detail.TipoImpositivo == rate {
			return detail
		}
	}

	return &doc.DetalleIVA{}
}

func findExemption(exemptions []*doc.DetalleExenta, cause string) *doc.DetalleExenta {
	for _, exemption := range exemptions {
		if exemption.CausaExencion == cause {
			return exemption
		}
	}

	return nil
}
