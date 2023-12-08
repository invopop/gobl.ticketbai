package doc

import (
	"errors"
	"time"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
)

// Factura contains the invoice info
type Factura struct {
	CabeceraFactura *CabeceraFactura
	DatosFactura    *DatosFactura
	TipoDesglose    *TipoDesglose
}

// CabeceraFactura contains info about the invoice header
type CabeceraFactura struct {
	NumFactura             string
	FechaExpedicionFactura string
	HoraExpedicionFactura  string
	FacturaSimplificada    string
	// FIXME: FacturasRectificativa
}

// DatosFactura contains info about the invoice description
// and totals
type DatosFactura struct {
	FechaOperacion      string
	DescripcionFactura  string
	DetallesFactura     *DetallesFactura
	ImporteTotalFactura string
	RetencionSoportada  string
	Claves              *Claves
}

// Claves contains a list of keys (min 1, max 3) of VAT types
type Claves struct {
	IDClave []IDClave
}

// IDClave is the key of a single VAT type
type IDClave struct {
	ClaveRegimenIvaOpTrascendencia string
}

func newCabeceraFactura(inv *bill.Invoice, ts time.Time) *CabeceraFactura {
	simplifiedInvoice := "N"
	if inv.Tax.ContainsTag(tax.TagSimplified) {
		simplifiedInvoice = "S"
	}

	// make sure TZ is correct
	ts = ts.In(location)
	issueDate := ts.Format("02-01-2006")
	issueTime := ts.Format("15:04:05")

	return &CabeceraFactura{
		NumFactura:             inv.Code,
		FechaExpedicionFactura: issueDate,
		HoraExpedicionFactura:  issueTime,
		FacturaSimplificada:    simplifiedInvoice,
	}
}

func newDatosFactura(inv *bill.Invoice) (*DatosFactura, error) {
	description, err := newDescription(inv.Notes)
	if err != nil {
		return nil, err
	}

	// This is only needed on Guipuzcoa and Alava, but Vizcaya documentation
	// states that it will be safely ignored so it will be added for everyone
	lineDetails := newDetallesFactura(inv)

	opDate := inv.OperationDate
	if opDate == nil {
		opDate = &inv.IssueDate
	}
	opDateStr := opDate.In(location).Format("02-01-2006")

	return &DatosFactura{
		FechaOperacion:      opDateStr,
		DescripcionFactura:  description,
		DetallesFactura:     lineDetails,
		ImporteTotalFactura: newImporteTotal(inv),
		RetencionSoportada:  newRetencionSoportada(inv),
		Claves:              &Claves{IDClave: newClaves(inv)},
	}, nil
}

func newDescription(notes []*cbc.Note) (string, error) {
	if notes == nil {
		return "", errors.New("missing general description of invoice")
	}

	for _, note := range notes {
		if note.Key == cbc.NoteKeyGeneral {
			return note.Text, nil
		}
	}

	return "", errors.New("missing general description of invoice")
}

func newImporteTotal(inv *bill.Invoice) string {
	totalWithDiscounts := inv.Totals.Total

	totalTaxes := num.MakeAmount(0, 2)
	for _, category := range inv.Totals.Taxes.Categories {
		if !category.Retained {
			totalTaxes = totalTaxes.Add(category.Amount)
		}
	}

	return totalWithDiscounts.Add(totalTaxes).String()
}

func newRetencionSoportada(inv *bill.Invoice) string {
	totalRetention := num.MakeAmount(0, 2)
	for _, category := range inv.Totals.Taxes.Categories {
		if category.Retained {
			totalRetention = totalRetention.Add(category.Amount)
		}
	}

	return totalRetention.String()
}

func newClaves(inv *bill.Invoice) []IDClave {
	claves := []IDClave{}

	if inv.Customer != nil && inv.Customer.TaxID.Country != "ES" {
		claves = append(claves, IDClave{
			ClaveRegimenIvaOpTrascendencia: "02",
		})
	}

	if hasSurchargedLines(inv) {
		claves = append(claves, IDClave{
			ClaveRegimenIvaOpTrascendencia: "51",
		})
	}

	if underSimplifiedRegime(inv) {
		claves = append(claves, IDClave{
			ClaveRegimenIvaOpTrascendencia: "52",
		})
	}

	if len(claves) == 0 {
		claves = append(claves, IDClave{
			ClaveRegimenIvaOpTrascendencia: "01",
		})
	}

	return claves
}

func hasSurchargedLines(inv *bill.Invoice) bool {
	for _, line := range inv.Lines {
		if line.Item.Key == es.ItemResale {
			return true
		}
	}
	return false
}

func underSimplifiedRegime(inv *bill.Invoice) bool {
	return inv.Tax != nil && inv.Tax.ContainsTag(es.TagSimplifiedScheme)
}