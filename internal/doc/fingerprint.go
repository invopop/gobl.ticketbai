package doc

import "github.com/invopop/gobl/cal"

// HuellaTBAI contains info about the Tickebai fingerprint
type HuellaTBAI struct {
	EncadenamientoFacturaAnterior *EncadenamientoFacturaAnterior `xml:",omitempty"`
	Software                      *Software
}

// EncadenamientoFacturaAnterior has the info of the previous
// invoice generated
type EncadenamientoFacturaAnterior struct {
	SerieFacturaAnterior               string
	NumFacturaAnterior                 string
	FechaExpedicionFacturaAnterior     string
	SignatureValueFirmaFacturaAnterior string
}

// FingerPrintConfig defines what is expected to produce
// a new fingerprint of a document.
type FingerprintConfig struct {
	License         string
	NIF             string
	SoftwareName    string
	SoftwareVersion string
	LastSeries      string
	LastCode        string
	LastIssueDate   cal.Date
	LastSignature   string
}

// Software used to generate the Ticketbai invoice
type Software struct {
	LicenciaTBAI          string `xml:",omitempty"`
	EntidadDesarrolladora *EntidadDesarrolladora
	Nombre                string
	Version               string
}

// EntidadDesarrolladora is the company that has developed the
// software that has created the Ticketbai invoice
type EntidadDesarrolladora struct {
	NIF string
}

func (doc *TicketBAI) buildHuellaTBAI(conf *FingerprintConfig) error {
	doc.HuellaTBAI = &HuellaTBAI{
		EncadenamientoFacturaAnterior: nil,
		Software: &Software{
			LicenciaTBAI: conf.License,
			EntidadDesarrolladora: &EntidadDesarrolladora{
				NIF: conf.NIF,
			},
			Nombre:  conf.SoftwareName,
			Version: conf.SoftwareVersion,
		},
	}
	if conf.LastCode != "" {
		dStr := conf.LastIssueDate.In(location).Format("02-01-2006")
		doc.HuellaTBAI.EncadenamientoFacturaAnterior = &EncadenamientoFacturaAnterior{
			SerieFacturaAnterior:               conf.LastSeries,
			NumFacturaAnterior:                 conf.LastCode,
			FechaExpedicionFacturaAnterior:     dStr,
			SignatureValueFirmaFacturaAnterior: conf.LastSignature,
		}
	}
	return nil
}
