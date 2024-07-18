package doc

import (
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
)

const (
	idTypeCodeTaxID = "02"
)

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

func newDestinatario(party *org.Party) (*IDDestinatario, error) {
	d := &IDDestinatario{
		ApellidosNombreRazonSocial: party.Name,
	}

	if partyTaxCountry(party) == "ES" {
		d.NIF = party.TaxID.Code.String()
	} else {
		d.IDOtro = otherIdentity(party)
	}
	if d.NIF == "" && d.IDOtro == nil {
		return nil, validationErr("customer with tax ID or other identity is required")
	}

	if len(party.Addresses) > 0 && party.Addresses[0].Code != "" {
		d.CodigoPostal = party.Addresses[0].Code
		d.Direccion = formatAddress(party.Addresses[0])
	}

	return d, nil
}

func otherIdentity(party *org.Party) *IDOtro {
	oid := new(IDOtro)
	if party.TaxID != nil {
		oid.CodigoPais = party.TaxID.Country.String()
	}

	if party.TaxID != nil && party.TaxID.Code != "" {
		oid.IDType = idTypeCodeTaxID
		oid.ID = party.TaxID.Code.String()
		return oid
	}

	r := tax.RegimeFor(l10n.ES)
	for _, it := range r.IdentityKeys {
		if id := org.IdentityForKey(party.Identities, it.Key); id != nil {
			oid.IDType = it.Map[es.KeyTicketBAIIDType].String()
			oid.ID = id.Code.String()
			return oid
		}
	}

	return nil
}

func partyTaxCountry(party *org.Party) l10n.CountryCode {
	if party != nil && party.TaxID != nil {
		return party.TaxID.Country
	}
	return ""
}
