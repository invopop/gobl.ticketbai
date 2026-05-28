package convert

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

var regime = tax.RegimeDefFor("ES")

// DetallesFactura contains a list of detail lines info
type DetallesFactura struct {
	IDDetalleFactura []IDDetalleFactura
}

// IDDetalleFactura contains info about a detail line of the invoice
type IDDetalleFactura struct {
	DescripcionDetalle string
	Cantidad           string
	ImporteUnitario    string // Without VAT
	Descuento          string
	ImporteTotal       string
}

func newDetallesFactura(gobl *bill.Invoice) *DetallesFactura {
	lines := []IDDetalleFactura{}
	for _, line := range gobl.Lines {
		if line.Item.Price == nil {
			continue
		}
		lines = append(lines, IDDetalleFactura{
			DescripcionDetalle: line.Item.Name,
			Cantidad:           line.Quantity.String(),
			ImporteUnitario:    line.Item.Price.Rescale(2).String(),
			Descuento:          calculateDiscounts(line).String(),
			ImporteTotal:       calculateTotal(line).Rescale(2).String(),
		})
	}

	return &DetallesFactura{
		IDDetalleFactura: lines,
	}
}

func calculateDiscounts(line *bill.Line) num.Amount {
	return line.Sum.Subtract(*line.Total)
}

func calculateTotal(line *bill.Line) num.Amount {
	taxes := calculateTaxes(line)

	return line.Total.Add(taxes)
}

func calculateTaxes(line *bill.Line) num.Amount {
	total := num.MakeAmount(0, 0)
	for _, t := range line.Taxes {
		if regime.CategoryDef(t.Category).Retained {
			continue
		}
		if t.Percent != nil {
			total = total.Add(t.Percent.Of(*line.Total))
		}
	}
	return total
}
