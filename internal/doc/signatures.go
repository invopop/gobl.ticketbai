package doc

import (
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/xmldsig"
)

func (doc *TicketBAI) sign(docID string, cert *xmldsig.Certificate, sigopts ...xmldsig.Option) error {
	data, err := doc.canonical()
	if err != nil {
		return err
	}

	sigopts = append(sigopts,
		xmldsig.WithDocID(docID),
		xmldsig.WithXAdES(XAdESConfig(doc.zone)),
		xmldsig.WithCertificate(cert),
		xmldsig.WithNamespace("T", ticketBAINamespace),
	)
	doc.Signature, err = xmldsig.Sign(data, sigopts...)
	if err != nil {
		return err
	}

	return nil
}

// SignatureValue provides quick access to the XML signatures final value,
// useful for inclusion in the database.
func (doc *TicketBAI) SignatureValue() string {
	if doc.Signature == nil {
		return ""
	}
	return doc.Signature.Value.Value
}

func (doc *TicketBAI) canonical() ([]byte, error) {
	buf, err := doc.buffer("", false)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// XAdESConfig returns the policies configuration for signing a TicketBAI doc
func XAdESConfig(zone l10n.Code) *xmldsig.XAdESConfig {
	if zone == es.ZoneBI {
		return &xmldsig.XAdESConfig{
			Role:        xmldsig.XAdESThirdParty,
			Description: "",
			Policy: &xmldsig.XAdESPolicyConfig{
				URL:         "https://www.batuz.eus/fitxategiak/batuz/ticketbai/sinadura_elektronikoaren_zehaztapenak_especificaciones_de_la_firma_electronica_v1_0.pdf",
				Description: "",
				Algorithm:   xmldsig.AlgDSigRSASHA256,
				Hash:        "Quzn98x3PMbSHwbUzaj5f5KOpiH0u8bvmwbbbNkO9Es=",
			},
		}
	}
	return nil
}
