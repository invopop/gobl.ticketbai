package doc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
)

// DetallesFactura constains a list of detail lines info
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
		total := calculateTotal(line)
		lines = append(lines, IDDetalleFactura{
			DescripcionDetalle: line.Item.Name,
			Cantidad:           line.Quantity.String(),
			ImporteUnitario:    line.Item.Price.Rescale(2).String(),
			Descuento:          calculateDiscounts(line).String(),
			ImporteTotal:       total.Rescale(2).String(),
		})
	}

	return &DetallesFactura{
		IDDetalleFactura: lines,
	}
}

func calculateTotal(line *bill.Line) num.Amount {
	subtotal := line.Item.Price.Multiply(line.Quantity)
	discount := calculateDiscounts(line)
	taxes := calculateTaxes(subtotal.Subtract(discount), line)

	return subtotal.Subtract(discount).Add(taxes)
}

func calculateTaxes(taxableAmount num.Amount, line *bill.Line) num.Amount {
	total := num.MakeAmount(0, 0)

	for _, tax := range line.Taxes {
		total = total.Add(tax.Percent.Of(taxableAmount))
	}

	return total
}
