package doc

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
)

func calculateDiscounts(line *bill.Line) num.Amount {
	return line.Sum.Subtract(line.Total)
}
