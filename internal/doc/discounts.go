package doc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
)

func calculateDiscounts(line *bill.Line) num.Amount {
	total := num.MakeAmount(0, 0)

	for _, discount := range line.Discounts {
		total = total.Add(discount.Amount)
	}

	return total
}
