package convert

import (
	"github.com/invopop/gobl/addons/es/tbai"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
)

const (
	idTypeCodeTaxID   = "02" // NIF-VAT (VIES)
	idTypeCodeForeign = "04" // Foreign identity document
)

var idTypeCodeMap = map[cbc.Key]string{
	org.IdentityKeyPassport: "03",
	org.IdentityKeyForeign:  idTypeCodeForeign,
	org.IdentityKeyResident: "05",
	org.IdentityKeyOther:    "06",
}

// Sujetos contains invoice parties info
type Sujetos struct {
	Emisor                          *Emisor
	Destinatarios                   *Destinatarios
	EmitidaPorTercerosODestinatario string
}

// Emisor contains info about the invoice supplier
type Emisor struct {
	NIF                        string
	ApellidosNombreRazonSocial string
}

// Destinatarios contains info about the invoice customers,
// Tickebai allows up to 100 customers but GOBL only allows one
// per invoice
type Destinatarios struct {
	IDDestinatario []*IDDestinatario
}

// IDDestinatario contains info about a single customer
type IDDestinatario struct {
	NIF                        string  `xml:",omitempty"`
	IDOtro                     *IDOtro `xml:",omitempty"`
	ApellidosNombreRazonSocial string
	CodigoPostal               string `xml:",omitempty"`
	Direccion                  string `xml:",omitempty"`
}

// IDOtro identifies a customer for a contry other than Spain
type IDOtro struct {
	CodigoPais string `xml:",omitempty"`
	IDType     string
	ID         string
}

func newEmisor(party *org.Party) *Emisor {
	return &Emisor{
		NIF:                        party.TaxID.Code.String(),
		ApellidosNombreRazonSocial: party.Name,
	}
}

func newDestinatario(party *org.Party) *IDDestinatario {
	d := &IDDestinatario{
		ApellidosNombreRazonSocial: party.Name,
	}

	if party.TaxID != nil && party.TaxID.Country == "ES" && party.TaxID.Code != "" {
		d.NIF = party.TaxID.Code.String()
	} else {
		d.IDOtro = otherIdentity(party)
	}
	if d.NIF == "" && d.IDOtro == nil {
		// Assume this is a B2C operation.
		return nil
	}

	if len(party.Addresses) > 0 && party.Addresses[0].Code != "" {
		d.CodigoPostal = party.Addresses[0].Code.String()
		d.Direccion = formatAddress(party.Addresses[0])
	}

	return d
}

func otherIdentity(party *org.Party) *IDOtro {
	oid := new(IDOtro)
	if party.TaxID != nil {
		oid.CodigoPais = party.TaxID.Country.String()
	}

	if party.TaxID != nil && party.TaxID.Code != "" {
		// EU customers get IDType=02 (NIF-VAT, validated against VIES);
		// non-EU customers' tax IDs (RFC, EIN, ...) are reported as
		// foreign identity documents (IDType=04). The TicketBAI gateway
		// rejects a non-EU tax ID submitted as NIF-VAT with B4_2000013.
		if l10n.Union(l10n.EU).HasMember(party.TaxID.Country.Code()) {
			oid.IDType = idTypeCodeTaxID
		} else {
			oid.IDType = idTypeCodeForeign
		}
		oid.ID = party.TaxID.Code.String()
		return oid
	}

	for _, id := range party.Identities {
		if id == nil || id.Code == "" {
			continue
		}
		code := id.Ext.Get(tbai.ExtKeyIdentityType).String()
		if code == "" {
			// Fallback to the legacy key map for documents not normalized
			// through the tbai addon.
			it, ok := idTypeCodeMap[id.Key]
			if !ok {
				continue
			}
			code = it
		}

		oid.IDType = code
		oid.ID = id.Code.String()
		if id.Country != "" {
			oid.CodigoPais = id.Country.String()
		}
		return oid
	}

	return nil
}

func partyCountry(party *org.Party) l10n.TaxCountryCode {
	if party == nil {
		return ""
	}
	if party.TaxID != nil && party.TaxID.Country != "" {
		return party.TaxID.Country
	}
	for _, id := range party.Identities {
		if id != nil && id.Country != "" {
			return l10n.TaxCountryCode(id.Country)
		}
	}
	return ""
}
