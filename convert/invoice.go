package convert

import (
	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
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

func newDescription(notes []*org.Note) (string, error) {
	for _, note := range notes {
		if note.Key == org.NoteKeyGeneral {
			return note.Text, nil
		}
	}
	return "", validationErr(`notes: missing note with key '%s'`, org.NoteKeyGeneral)
}

func newImporteTotal(inv *bill.Invoice) string {
	totalWithDiscounts := inv.Totals.Total

	totalTaxes := num.MakeAmount(0, 2)
	if inv.Totals.Taxes != nil {
		for _, category := range inv.Totals.Taxes.Categories {
			if !category.Retained {
				totalTaxes = totalTaxes.Add(category.Amount)
			}
		}
	}

	return totalWithDiscounts.Add(totalTaxes).String()
}

func newRetencionSoportada(inv *bill.Invoice) string {
	totalRetention := num.MakeAmount(0, 2)
	if inv.Totals.Taxes != nil {
		for _, category := range inv.Totals.Taxes.Categories {
			if category.Retained {
				totalRetention = totalRetention.Add(category.Amount)
			}
		}
	}

	return totalRetention.String()
}

func newClaves(inv *bill.Invoice) []IDClave {
	// Preferred: collect unique codes from the es-tbai-regime extension
	// set per VAT tax combo by the es-tbai-v1 addon normalizer. This is
	// the path that respects an explicit ClaveRegimenIvaOpTrascendencia
	// the caller may have set.
	if codes := collectRegimeCodes(inv); len(codes) > 0 {
		claves := make([]IDClave, 0, len(codes))
		for _, c := range codes {
			claves = append(claves, IDClave{ClaveRegimenIvaOpTrascendencia: c})
		}
		return claves
	}

	return legacyClaves(inv)
}

// collectRegimeCodes returns the distinct ClaveRegimenIvaOpTrascendencia
// codes carried by the invoice's VAT rate totals via the es-tbai-regime
// extension, preserving the order of first appearance.
func collectRegimeCodes(inv *bill.Invoice) []string {
	if inv.Totals == nil || inv.Totals.Taxes == nil {
		return nil
	}
	vat := inv.Totals.Taxes.Category(tax.CategoryVAT)
	if vat == nil {
		return nil
	}
	seen := make(map[string]bool, len(vat.Rates))
	codes := make([]string, 0, len(vat.Rates))
	for _, rate := range vat.Rates {
		c := rate.Ext.Get(tbai.ExtKeyRegime).String()
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		codes = append(codes, c)
	}
	return codes
}

// legacyClaves is the original invoice-level inference of
// ClaveRegimenIvaOpTrascendencia, kept as a fallback for callers that
// build TicketBAI documents from GOBL invoices that have not been
// normalized by the es-tbai-v1 addon. New code should rely on the
// es-tbai-regime extension instead.
func legacyClaves(inv *bill.Invoice) []IDClave {
	claves := []IDClave{}

	if inv.Customer != nil && partyCountry(inv.Customer) != "ES" {
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
		Codigo: p.Ext.Get(tbai.ExtKeyCorrection).String(),
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

// hasSurchargedLines is part of the legacy ClaveRegimenIvaOpTrascendencia
// inference. Detection now happens at addon-normalization time and is
// reflected in the es-tbai-regime extension; see legacyClaves.
func hasSurchargedLines(inv *bill.Invoice) bool {
	if inv.Totals == nil || inv.Totals.Taxes == nil {
		return false
	}
	vat := inv.Totals.Taxes.Category(tax.CategoryVAT)
	if vat == nil {
		return false
	}

	for _, rate := range vat.Rates {
		if rate.Ext.Get(tbai.ExtKeyProduct) == "resale" {
			return true
		}
	}

	return false
}

// underSimplifiedRegime is part of the legacy regime inference; see
// legacyClaves.
func underSimplifiedRegime(inv *bill.Invoice) bool {
	return inv.HasTags(es.TagSimplifiedScheme)
}
