package doc

import (
	"strings"

	"github.com/invopop/gobl/bill"
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
	PrestacionServicios *PrestacionServicios
	Entrega             *Entrega
}

// PrestacionServicios means that there is no exchange of goods
type PrestacionServicios struct {
	Sujeta   *Sujeta
	NoSujeta *NoSujeta
}

// Entrega means that there is an exchange of goods
type Entrega struct {
	Sujeta   *Sujeta
	NoSujeta *NoSujeta
}

// Sujeta means that some amount is liable to taxes (VAT, equivalence surcharge)
type Sujeta struct {
	Exenta   *Exenta
	NoExenta *NoExenta
}

// Exenta contains the info of the amounts that even being liable to taxes, are
// under 0% rate due to different reasons
type Exenta struct {
	DetalleExenta []DetalleExenta
}

// DetalleExenta details about 0% taxed amounts
type DetalleExenta struct {
	CausaExencion string
	BaseImponible string
}

// NoExenta list the amounts that a liable to taxes (VAT, equivalence surcharge) other than 0%
type NoExenta struct {
	DetalleNoExenta []DetalleNoExenta
}

// DetalleNoExenta details about non 0% taxes amounts
type DetalleNoExenta struct {
	TipoNoExenta string
	DesgloseIVA  *DesgloseIVA
}

// DesgloseIVA list of VAT details
type DesgloseIVA struct {
	DetalleIVA []DetalleIVA
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
	DetalleNoSujeta []DetalleNoSujeta
}

// DetalleNoSujeta contails details about the not liable amount
type DetalleNoSujeta struct {
	Causa   string
	Importe string
}

const goods = "goods"

type taxInfo struct {
	simplifiedRegime bool
	reverseCharge    bool
	customerRates    bool
}

func newTipoDesglose(gobl *bill.Invoice) *TipoDesglose {
	desglose := &TipoDesglose{}

	taxInfo := taxInfo{}
	if gobl.Tax != nil {
		for _, scheme := range gobl.Tax.Tags {
			switch scheme {
			case es.TagSimplifiedScheme:
				taxInfo.simplifiedRegime = true
			case tax.TagReverseCharge:
				taxInfo.reverseCharge = true
			case tax.TagCustomerRates:
				taxInfo.customerRates = true
			}
		}
	}

	if gobl.Customer == nil || gobl.Customer.TaxID.Country == l10n.ES {
		desglose.DesgloseFactura = &DesgloseFactura{
			NoSujeta: newNoSujeta(gobl.Lines, taxInfo),
			Sujeta:   newSujeta(gobl.Lines, taxInfo),
		}
	} else {
		desglose.DesgloseTipoOperacion = &DesgloseTipoOperacion{}

		goodsLines := filterGoodsLines(gobl)
		if len(goodsLines) > 0 {
			desglose.DesgloseTipoOperacion.Entrega = &Entrega{
				NoSujeta: newNoSujeta(goodsLines, taxInfo),
				Sujeta:   newSujeta(goodsLines, taxInfo),
			}
		}

		serviceLines := filterServiceLines(gobl)
		if len(serviceLines) > 0 {
			desglose.DesgloseTipoOperacion.PrestacionServicios = &PrestacionServicios{
				NoSujeta: newNoSujeta(serviceLines, taxInfo),
				Sujeta:   newSujeta(serviceLines, taxInfo),
			}
		}
	}

	return desglose
}

func filterGoodsLines(gobl *bill.Invoice) []*bill.Line {
	lines := []*bill.Line{}

	for _, line := range gobl.Lines {
		if line.Item.Key == es.ItemGoods {
			lines = append(lines, line)
		}
	}

	return lines
}

func filterServiceLines(gobl *bill.Invoice) []*bill.Line {
	lines := []*bill.Line{}

	for _, line := range gobl.Lines {
		if line.Item.Key != es.ItemGoods {
			lines = append(lines, line)
		}
	}

	return lines
}

func newNoSujeta(lines []*bill.Line, taxInfo taxInfo) *NoSujeta {
	sum := sumNoSujetaAmount(lines)

	if sum.IsZero() {
		return nil
	}

	cause := "OT"
	if taxInfo.customerRates {
		cause = "RL"
	}

	return &NoSujeta{
		DetalleNoSujeta: []DetalleNoSujeta{
			{
				Causa:   cause,
				Importe: sum.Rescale(2).String(),
			},
		},
	}
}

func sumNoSujetaAmount(lines []*bill.Line) num.Amount {
	sum := num.MakeAmount(0, 2)

	for _, line := range lines {
		withoutTaxes := true
		for _, tax := range line.Taxes {
			if tax.Category != es.TaxCategoryIRPF {
				withoutTaxes = false
			}
		}

		if withoutTaxes {
			sum = sum.Add(line.Total)
		}
	}

	return sum
}

func newSujeta(lines []*bill.Line, taxInfo taxInfo) *Sujeta {
	details, detailsExempted := buildDetails(lines, taxInfo)

	if len(details) == 0 && len(detailsExempted) == 0 {
		return nil
	}

	var noExenta *NoExenta
	if len(details) > 0 {
		noExenta = &NoExenta{
			DetalleNoExenta: []DetalleNoExenta{
				{
					TipoNoExenta: nonExemptedType(taxInfo),
					DesgloseIVA:  &DesgloseIVA{DetalleIVA: details},
				},
			},
		}
	}

	var exenta *Exenta
	if len(detailsExempted) > 0 {
		exenta = &Exenta{
			DetalleExenta: detailsExempted,
		}
	}

	return &Sujeta{
		NoExenta: noExenta,
		Exenta:   exenta,
	}
}

func nonExemptedType(taxInfo taxInfo) string {
	if taxInfo.reverseCharge {
		return "S2"
	}

	return "S1"
}

func buildDetails(lines []*bill.Line, taxInfo taxInfo) ([]DetalleIVA, []DetalleExenta) {
	exempted, nonExempted, surcharged := sumAmountsPerType(lines)

	exemptedList := []DetalleExenta{}
	for cause, sum := range exempted {
		exemptedList = append(exemptedList, DetalleExenta{
			CausaExencion: cause,
			BaseImponible: sum.amount.Rescale(2).String(),
		})
	}

	detailList := []DetalleIVA{}
	for _, sum := range nonExempted {
		detail := DetalleIVA{
			BaseImponible:  sum.amount.Rescale(2).String(),
			TipoImpositivo: formatPercent(sum.percent),
			CuotaImpuesto:  sum.percent.Of(sum.amount).Rescale(2).String(),
		}

		if !sum.surcharge.IsZero() {
			detail.TipoRecargoEquivalencia = formatPercent(sum.surcharge)
			detail.CuotaRecargoEquivalencia = sum.surcharge.Of(sum.amount).Rescale(2).String()
		}

		if taxInfo.simplifiedRegime {
			detail.OperacionEnRecargoDeEquivalenciaORegimenSimplificado = "S"
		}

		detailList = append(detailList, detail)
	}

	for _, sum := range surcharged {
		detailList = append(detailList, DetalleIVA{
			BaseImponible:  sum.amount.Rescale(2).String(),
			TipoImpositivo: formatPercent(sum.percent),
			CuotaImpuesto:  sum.percent.Of(sum.amount).Rescale(2).String(),
			OperacionEnRecargoDeEquivalenciaORegimenSimplificado: "S",
		})
	}

	return detailList, exemptedList
}

type sumDetail struct {
	amount    num.Amount
	percent   num.Percentage
	surcharge num.Percentage
}

func sumAmountsPerType(lines []*bill.Line) (map[string]sumDetail, map[string]sumDetail, map[string]sumDetail) {
	exempted := make(map[string]sumDetail)
	nonExempted := make(map[string]sumDetail)
	surcharged := make(map[string]sumDetail)

	for _, line := range lines {
		discount := calculateDiscounts(line)
		// TODO: Handle charges
		taxableAmount := line.Item.Price.Multiply(line.Quantity).Subtract(discount)
		lineSurcharged := line.Item.Key == es.ItemResale

		for _, t := range line.Taxes {
			if t.Category == tax.CategoryVAT && t.Rate == tax.RateExempt {
				exempted = updateAmount(
					exempted,
					t.Ext[es.ExtKeyTBAIExemption].String(),
					taxableAmount,
					num.MakePercentage(0, 0),
					surcharge(t),
				)
			} else if t.Category == tax.CategoryVAT && lineSurcharged {
				surcharged = updateAmount(
					surcharged,
					taxKey(t),
					taxableAmount,
					*t.Percent,
					surcharge(t),
				)
			} else if t.Category == tax.CategoryVAT {
				nonExempted = updateAmount(
					nonExempted,
					taxKey(t),
					taxableAmount,
					*t.Percent,
					surcharge(t),
				)
			}
		}
	}

	return exempted, nonExempted, surcharged
}

func formatPercent(percent num.Percentage) string {
	maybeNegative := percent.Rescale(4).Multiply(num.MakeAmount(100, 0)).Rescale(2).String()
	if strings.Contains(maybeNegative, "-") {
		return strings.Replace(maybeNegative, "-", "", -1)
	}

	return maybeNegative
}

func taxKey(tax *tax.Combo) string {
	key := tax.Percent.String()
	if tax.Surcharge != nil {
		key = key + "+" + tax.Surcharge.String()
	}

	return key
}

func updateAmount(
	totals map[string]sumDetail,
	key string,
	taxableAmount num.Amount,
	percentage num.Percentage,
	surcharge num.Percentage,
) map[string]sumDetail {
	total, found := totals[key]
	if !found {
		totals[key] = sumDetail{percent: percentage, amount: taxableAmount, surcharge: surcharge}
	} else {
		total.amount = total.amount.Add(taxableAmount)
		totals[key] = total
	}

	return totals
}

func surcharge(tax *tax.Combo) num.Percentage {
	var surcharge num.Percentage
	if tax.Surcharge != nil {
		surcharge = *tax.Surcharge
	}
	return surcharge
}
