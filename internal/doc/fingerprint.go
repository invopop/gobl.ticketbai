package doc

// HuellaTBAI contains info about the Tickebai fingerprint
type HuellaTBAI struct {
	EncadenamientoFacturaAnterior *EncadenamientoFacturaAnterior `xml:",omitempty"`
	Software                      *Software
}

// EncadenamientoFacturaAnterior has the info of the previous invoice generated
type EncadenamientoFacturaAnterior struct {
	SerieFacturaAnterior               string `xml:",omitempty"`
	NumFacturaAnterior                 string
	FechaExpedicionFacturaAnterior     string
	SignatureValueFirmaFacturaAnterior string
}

// FingerprintConfig defines what is expected to produce a new fingerprint of a
// document.
type FingerprintConfig struct {
	License         string
	NIF             string
	SoftwareName    string
	SoftwareVersion string
	LastSeries      string
	LastCode        string
	LastIssueDate   string
	LastSignature   string
}

// Software used to generate the Ticketbai invoice
type Software struct {
	LicenciaTBAI          string
	EntidadDesarrolladora *EntidadDesarrolladora
	Nombre                string
	Version               string
}

// EntidadDesarrolladora is the company that has developed the software that has
// created the Ticketbai invoice
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
		doc.HuellaTBAI.EncadenamientoFacturaAnterior = &EncadenamientoFacturaAnterior{
			SerieFacturaAnterior:               conf.LastSeries,
			NumFacturaAnterior:                 conf.LastCode,
			FechaExpedicionFacturaAnterior:     conf.LastIssueDate,
			SignatureValueFirmaFacturaAnterior: trunc(conf.LastSignature, 100),
		}
	}

	return nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
