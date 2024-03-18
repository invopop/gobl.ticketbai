package doc

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
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
	IDDestinatario []IDDestinatario
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

func newDestinatario(party *org.Party) IDDestinatario {
	destinatario := IDDestinatario{
		ApellidosNombreRazonSocial: party.Name,
	}

	if party.TaxID.Country == "ES" {
		destinatario.NIF = party.TaxID.Code.String()
	} else {
		destinatario.IDOtro = &IDOtro{
			CodigoPais: party.TaxID.Country.String(),
			IDType:     taxDocumentType(party).String(),
			ID:         party.TaxID.Code.String(),
		}
	}

	if len(party.Addresses) > 0 && party.Addresses[0].Code != "" {
		destinatario.CodigoPostal = party.Addresses[0].Code
		destinatario.Direccion = formatAddress(party.Addresses[0])
	}

	return destinatario
}

func taxDocumentType(party *org.Party) cbc.Code {
	r := tax.RegimeFor(l10n.ES)

	t := party.TaxID.Type
	if t == "" {
		t = es.TaxIdentityTypeFiscal
	}

	for _, it := range r.IdentityTypeKeys {
		if it.Key == t {
			return it.Map[es.KeyTicketBAIIDType]
		}
	}

	return ""
}
