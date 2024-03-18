package doc

import (
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/xmldsig"
)

// SignerRoles defined in the TicketBAI spec
const (
	XAdESSupplier   xmldsig.XAdESSignerRole = "Supplier"
	XAdESCustomer   xmldsig.XAdESSignerRole = "Customer"
	XAdESThirdParty xmldsig.XAdESSignerRole = "Thirdparty"
)

func newSignature(doc any, docID string, zone l10n.Code, role IssuerRole, cert *xmldsig.Certificate, opts ...xmldsig.Option) (*xmldsig.Signature, error) {
	data, err := toBytesCanonical(doc)
	if err != nil {
		return nil, err
	}

	opts = append(opts,
		xmldsig.WithDocID(docID),
		xmldsig.WithXAdES(XAdESConfig(zone, signerRole(role))),
		xmldsig.WithCertificate(cert),
		xmldsig.WithNamespace("T", ticketBAIEmisionNamespace),
	)

	return xmldsig.Sign(data, opts...)
}

// SignatureValue provides quick access to the XML signatures final value,
// useful for inclusion in the database.
func (doc *TicketBAI) SignatureValue() string {
	if doc.Signature == nil {
		return ""
	}
	return doc.Signature.Value.Value
}

func signerRole(role IssuerRole) xmldsig.XAdESSignerRole {
	switch role {
	case IssuerRoleSupplier:
		return XAdESSupplier
	case IssuerRoleCustomer:
		return XAdESCustomer
	case IssuerRoleThirdParty:
		return XAdESThirdParty
	default:
		return ""
	}
}

// XAdESConfig returns the policies configuration for signing a TicketBAI doc
func XAdESConfig(zone l10n.Code, role xmldsig.XAdESSignerRole) *xmldsig.XAdESConfig {
	if zone == ZoneBI {
		return &xmldsig.XAdESConfig{
			Role:        role,
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
