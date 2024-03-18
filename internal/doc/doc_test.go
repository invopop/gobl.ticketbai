package doc_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/regimes/es"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceConversion(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	require.NoError(t, err)
	role := doc.IssuerRoleThirdParty

	t.Run("fail when missing zone", func(t *testing.T) {
		inv, _ := test.LoadInvoice("sample-invoice.json")

		_, err := doc.NewTicketBAI(inv, ts, role, l10n.Code(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "zone is required")
	})

	t.Run("should have the right version", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")

		invoice, err := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		require.NoError(t, err)
		assert.Equal(t, "1.2", invoice.Cabecera.IDVersionTBAI)
	})

	t.Run("should contain info about the supplier", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Supplier.TaxID.Code = "X34789654"
		goblInvoice.Supplier.Name = "Fake Company SL"

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		assert.Equal(t, "Fake Company SL", invoice.Sujetos.Emisor.ApellidosNombreRazonSocial)
		assert.Equal(t, "X34789654", invoice.Sujetos.Emisor.NIF)
	})

	t.Run("should contain the issuer role code", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, doc.IssuerRoleCustomer, doc.ZoneBI)

		assert.Equal(t, "D", invoice.Sujetos.EmitidaPorTercerosODestinatario)
	})

	t.Run("should contain info about national customer", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = &tax.Identity{Country: "ES", Code: "17654245G"}
		goblInvoice.Customer.Name = "Spanish Co SL"
		goblInvoice.Customer.Addresses[0].Code = "50250"
		goblInvoice.Customer.Addresses[0].PostOfficeBox = "PO-745"

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		assert.Equal(t, "17654245G", invoice.Sujetos.Destinatarios.IDDestinatario[0].NIF)
		assert.Equal(t, "Spanish Co SL", invoice.Sujetos.Destinatarios.IDDestinatario[0].ApellidosNombreRazonSocial)
		assert.Equal(t, "50250", invoice.Sujetos.Destinatarios.IDDestinatario[0].CodigoPostal)
		assert.Contains(t, invoice.Sujetos.Destinatarios.IDDestinatario[0].Direccion, "PO-745")
	})

	t.Run("should contain the right id for abroad customers", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = &tax.Identity{Country: "GB", Code: "PP-123456-S"}
		goblInvoice.Customer.Name = "Abroad Co LLC"

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		assert.Equal(t, "GB", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.CodigoPais)
		assert.Equal(t, "PP-123456-S", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.ID)
		assert.Equal(t, "02", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.IDType)
		assert.Equal(t, "Abroad Co LLC", invoice.Sujetos.Destinatarios.IDDestinatario[0].ApellidosNombreRazonSocial)
	})

	t.Run("should change the document type from the default (02) if stated", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = &tax.Identity{
			Country: "GB", Code: "PP-123456-S", Type: es.TaxIdentityTypeResident,
		}

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		assert.Equal(t, "05", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.IDType)
	})

	t.Run("should allow having no customer (useful for simplied invoices)", func(t *testing.T) {
		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer = nil

		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)

		assert.Nil(t, invoice.Sujetos.Destinatarios)
	})
}
