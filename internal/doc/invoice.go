package doc

import (
	"github.com/invopop/gobl/addons/es/tbai"
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
	SerieFactura                    string `xml:",omitempty"`
	NumFactura                      string
	FechaExpedicionFactura          string
	HoraExpedicionFactura           string
	FacturaSimplificada             string
	FacturaRectificativa            *FacturaRectificativa            `xml:",omitempty"`
	FacturasRectificadasSustituidas *FacturasRectificadasSustituidas `xml:",omitempty"`
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

// FacturaRectificativa contains the info a corrective invoice
type FacturaRectificativa struct {
	Codigo string
	Tipo   string
}

// FacturasRectificadasSustituidas contains the info of all the invoices corrected or substituted in
// a corrective invoice
type FacturasRectificadasSustituidas struct {
	IDFacturaRectificadaSustituida []*IDFacturaRectificadaSustituida
}

// IDFacturaRectificadaSustituida contains the info of a single invoice corrected or substituted in
// a corrective invoice
type IDFacturaRectificadaSustituida struct {
	SerieFactura           string
	NumFactura             string
	FechaExpedicionFactura string
}

func newCabeceraFactura(inv *bill.Invoice) *CabeceraFactura {
	simplifiedInvoice := "N"
	if inv.HasTags(tax.TagSimplified) {
		simplifiedInvoice = "S"
	}

	return &CabeceraFactura{
		SerieFactura:                    inv.Series.String(),
		NumFactura:                      inv.Code.String(),
		FacturaSimplificada:             simplifiedInvoice,
		FacturaRectificativa:            newFacturaRectificativa(inv),
		FacturasRectificadasSustituidas: newFacturasRectificadasSustituidas(inv),
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
	opDateStr := formatDate(opDate)

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
	for _, note := range notes {
		if note.Key == cbc.NoteKeyGeneral {
			return note.Text, nil
		}
	}
	return "", validationErr(`notes: missing note with key '%s'`, cbc.NoteKeyGeneral)
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

	if inv.Customer != nil && partyTaxCountry(inv.Customer) != "ES" {
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

func newFacturaRectificativa(inv *bill.Invoice) *FacturaRectificativa {
	if len(inv.Preceding) == 0 {
		return nil
	}

	p := inv.Preceding[0]

	return &FacturaRectificativa{
		Codigo: p.Ext[tbai.ExtKeyCorrection].String(),
		Tipo:   CorrectiveTypeDifferences, // Only differences are supported for now
	}
}

func newFacturasRectificadasSustituidas(inv *bill.Invoice) *FacturasRectificadasSustituidas {
	if inv.Preceding == nil {
		return nil
	}

	p := inv.Preceding[0]

	return &FacturasRectificadasSustituidas{
		IDFacturaRectificadaSustituida: []*IDFacturaRectificadaSustituida{
			{
				SerieFactura:           p.Series.String(),
				NumFactura:             p.Code.String(),
				FechaExpedicionFactura: formatDate(p.IssueDate),
			},
		},
	}
}

func hasSurchargedLines(inv *bill.Invoice) bool {
	vat := inv.Totals.Taxes.Category(tax.CategoryVAT)
	if vat == nil {
		return false
	}

	for _, rate := range vat.Rates {
		if rate.Ext[tbai.ExtKeyProduct] == "resale" {
			return true
		}
	}

	return false
}

func underSimplifiedRegime(inv *bill.Invoice) bool {
	return inv.HasTags(es.TagSimplifiedScheme)
}
