package doc

import (
	"strings"

	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
)

// TipoDesglose contains info about the taxes breakdown of
// an invoice
type TipoDesglose struct {
	DesgloseFactura       *DesgloseFactura
	DesgloseTipoOperacion *DesgloseTipoOperacion
}

// DesgloseFactura contains taxes breakdown if customer is from Spain or there
// is no customer
type DesgloseFactura struct {
	Sujeta   *Sujeta
	NoSujeta *NoSujeta
}

// DesgloseTipoOperacion contains taxes breakdown if customer is from abroad
type DesgloseTipoOperacion struct {
	PrestacionServicios *DesgloseFactura
	Entrega             *DesgloseFactura
}

// Sujeta means that some amount is liable to taxes (VAT, equivalence surcharge)
type Sujeta struct {
	Exenta   *Exenta
	NoExenta *NoExenta
}

// Exenta contains the info of the amounts that even being liable to taxes, are
// under 0% rate due to different reasons
type Exenta struct {
	DetalleExenta []*DetalleExenta
}

// DetalleExenta details about 0% taxed amounts
type DetalleExenta struct {
	CausaExencion string
	BaseImponible string
}

// NoExenta list the amounts that a liable to taxes (VAT, equivalence surcharge) other than 0%
type NoExenta struct {
	DetalleNoExenta []*DetalleNoExenta
}

// DetalleNoExenta details about non 0% taxes amounts
type DetalleNoExenta struct {
	TipoNoExenta string
	DesgloseIVA  *DesgloseIVA
}

// DesgloseIVA list of VAT details
type DesgloseIVA struct {
	DetalleIVA []*DetalleIVA
}

// DetalleIVA contains details of VAT / equivalence surcharge taxes
type DetalleIVA struct {
	BaseImponible                                        string
	TipoImpositivo                                       string
	CuotaImpuesto                                        string
	TipoRecargoEquivalencia                              string `xml:",omitempty"`
	CuotaRecargoEquivalencia                             string `xml:",omitempty"`
	OperacionEnRecargoDeEquivalenciaORegimenSimplificado string `xml:",omitempty"`
}

// NoSujeta means that some part of the invoice is not liable to VAT
type NoSujeta struct {
	DetalleNoSujeta []*DetalleNoSujeta
}

// DetalleNoSujeta contains details about the not liable amount
type DetalleNoSujeta struct {
	Causa   string
	Importe num.Amount
}

type taxInfo struct {
	simplifiedRegime bool
	reverseCharge    bool
	customerRates    bool
}

func newTipoDesglose(gobl *bill.Invoice) *TipoDesglose {
	catTotal := gobl.Totals.Taxes.Category(tax.CategoryVAT)
	if catTotal == nil {
		return nil
	}
	taxInfo := newTaxInfo(gobl)

	desglose := &TipoDesglose{}

	if gobl.Customer == nil || partyTaxCountry(gobl.Customer) == l10n.ES.Tax() {
		desglose.DesgloseFactura = newDesgloseFactura(taxInfo, catTotal.Rates)
	} else {
		goods, services := splitByTBAIProduct(catTotal.Rates)

		desglose.DesgloseTipoOperacion = &DesgloseTipoOperacion{
			Entrega:             newDesgloseFactura(taxInfo, goods),
			PrestacionServicios: newDesgloseFactura(taxInfo, services),
		}
	}

	return desglose
}

func newDesgloseFactura(taxInfo taxInfo, rates []*tax.RateTotal) *DesgloseFactura {
	if len(rates) == 0 {
		return nil
	}

	df := &DesgloseFactura{
		NoSujeta: &NoSujeta{},
		Sujeta: &Sujeta{
			Exenta:   &Exenta{},
			NoExenta: &NoExenta{},
		},
	}

	for _, rate := range rates {
		if taxInfo.isNoSujeta(rate) {
			df.NoSujeta.appendDetalle(&DetalleNoSujeta{
				Causa:   taxInfo.causaNoSujeta(rate),
				Importe: rate.Base,
			})
		} else if taxInfo.isExenta(rate) {
			df.Sujeta.Exenta.appendDetalle(&DetalleExenta{
				CausaExencion: rate.Ext[tbai.ExtKeyExemption].String(),
				BaseImponible: rate.Base.Rescale(2).String(),
			})
		} else {
			dne := df.Sujeta.NoExenta.appendDetalle(&DetalleNoExenta{
				TipoNoExenta: taxInfo.nonExemptedType(),
				DesgloseIVA:  &DesgloseIVA{},
			})

			diva := newDetalleIVA(taxInfo, rate)

			dne.DesgloseIVA.appendDetalle(diva)
		}
	}

	return df.prune()
}

func splitByTBAIProduct(rates []*tax.RateTotal) (goods, services []*tax.RateTotal) {
	for _, rate := range rates {
		if rate.Ext[tbai.ExtKeyProduct] == "goods" {
			goods = append(goods, rate)
		} else {
			services = append(services, rate)
		}
	}

	return goods, services
}

func (df *DesgloseFactura) prune() *DesgloseFactura {
	if len(df.NoSujeta.DetalleNoSujeta) == 0 {
		df.NoSujeta = nil
	}
	if len(df.Sujeta.Exenta.DetalleExenta) == 0 {
		df.Sujeta.Exenta = nil
	}
	if len(df.Sujeta.NoExenta.DetalleNoExenta) == 0 {
		df.Sujeta.NoExenta = nil
	}
	if df.Sujeta.Exenta == nil && df.Sujeta.NoExenta == nil {
		df.Sujeta = nil
	}

	return df
}

func (ns *NoSujeta) appendDetalle(d *DetalleNoSujeta) *DetalleNoSujeta {
	for _, e := range ns.DetalleNoSujeta {
		if e.Causa == d.Causa {
			e.Importe = e.Importe.Add(d.Importe)
			return e
		}
	}
	ns.DetalleNoSujeta = append(ns.DetalleNoSujeta, d)
	return d
}

func (e *Exenta) appendDetalle(d *DetalleExenta) *DetalleExenta {
	e.DetalleExenta = append(e.DetalleExenta, d)
	return d
}

func (ne *NoExenta) appendDetalle(d *DetalleNoExenta) *DetalleNoExenta {
	for _, e := range ne.DetalleNoExenta {
		if e.TipoNoExenta == d.TipoNoExenta {
			return e
		}
	}
	ne.DetalleNoExenta = append(ne.DetalleNoExenta, d)
	return d
}

func (di *DesgloseIVA) appendDetalle(d *DetalleIVA) *DetalleIVA {
	di.DetalleIVA = append(di.DetalleIVA, d)
	return d
}

func newDetalleIVA(taxInfo taxInfo, rate *tax.RateTotal) *DetalleIVA {
	diva := &DetalleIVA{
		BaseImponible:  rate.Base.Rescale(2).String(),
		TipoImpositivo: formatPercent(*rate.Percent),
		CuotaImpuesto:  rate.Amount.Rescale(2).String(),
	}

	if rate.Surcharge != nil {
		diva.TipoRecargoEquivalencia = formatPercent(rate.Surcharge.Percent)
		diva.CuotaRecargoEquivalencia = rate.Surcharge.Amount.Rescale(2).String()
	}

	if taxInfo.simplifiedRegime || rate.Ext[tbai.ExtKeyProduct] == "resale" {
		diva.OperacionEnRecargoDeEquivalenciaORegimenSimplificado = "S"
	}

	return diva
}

func formatPercent(percent num.Percentage) string {
	maybeNegative := percent.Amount().Rescale(2).String()
	if strings.Contains(maybeNegative, "-") {
		return strings.Replace(maybeNegative, "-", "", -1)
	}

	return maybeNegative
}

func newTaxInfo(gobl *bill.Invoice) taxInfo {
	return taxInfo{
		simplifiedRegime: gobl.HasTags(es.TagSimplifiedScheme),
		reverseCharge:    gobl.HasTags(tax.TagReverseCharge),
		customerRates:    gobl.HasTags(tax.TagCustomerRates),
	}
}

func (t taxInfo) nonExemptedType() string {
	if t.reverseCharge {
		return "S2"
	}

	return "S1"
}

var notSubjectExemptionCodes = []cbc.Code{"OT", "RL"}

func (t taxInfo) isNoSujeta(r *tax.RateTotal) bool {
	if t.customerRates {
		return true
	}
	return r.Percent == nil && r.Ext[tbai.ExtKeyExemption].Code().In(notSubjectExemptionCodes...)
}

func (t taxInfo) causaNoSujeta(r *tax.RateTotal) string {
	if t.customerRates {
		return "RL"
	}
	return r.Ext[tbai.ExtKeyExemption].String()
}

func (taxInfo) isExenta(r *tax.RateTotal) bool {
	return r.Percent == nil && !r.Ext[tbai.ExtKeyExemption].Code().In(notSubjectExemptionCodes...)
}
