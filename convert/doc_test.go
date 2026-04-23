package convert_test

import (
	"os"
	"testing"
	"time"

	"github.com/nbio/xml"

	"github.com/invopop/gobl.ticketbai/convert"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceConversion(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	require.NoError(t, err)
	role := convert.IssuerRoleThirdParty

	t.Run("fail when missing zone", func(t *testing.T) {
		inv := test.LoadInvoice("sample-invoice.json")

		_, err := convert.NewTicketBAI(inv, ts, role, l10n.Code(""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "zone is required")
	})

	t.Run("should have the right version", func(t *testing.T) {
		inv := test.LoadInvoice("sample-invoice.json")
		invoice, err := convert.NewTicketBAI(inv, ts, role, convert.ZoneBI)

		require.NoError(t, err)
		assert.Equal(t, "1.2", invoice.Cabecera.IDVersionTBAI)
	})

	t.Run("should contain info about the supplier", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Supplier.TaxID.Code = "X34789654"
		goblInvoice.Supplier.Name = "Fake Company SL"

		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)

		assert.Equal(t, "Fake Company SL", invoice.Sujetos.Emisor.ApellidosNombreRazonSocial)
		assert.Equal(t, "X34789654", invoice.Sujetos.Emisor.NIF)
	})

	t.Run("should contain the issuer role code", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")

		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, convert.IssuerRoleCustomer, convert.ZoneBI)

		assert.Equal(t, "D", invoice.Sujetos.EmitidaPorTercerosODestinatario)
	})

	t.Run("should contain info about national customer", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = &tax.Identity{Country: "ES", Code: "17654245G"}
		goblInvoice.Customer.Name = "Spanish Co SL"
		goblInvoice.Customer.Addresses[0].Code = "50250"
		goblInvoice.Customer.Addresses[0].PostOfficeBox = "PO-745"

		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)

		assert.Equal(t, "17654245G", invoice.Sujetos.Destinatarios.IDDestinatario[0].NIF)
		assert.Equal(t, "Spanish Co SL", invoice.Sujetos.Destinatarios.IDDestinatario[0].ApellidosNombreRazonSocial)
		assert.Equal(t, "50250", invoice.Sujetos.Destinatarios.IDDestinatario[0].CodigoPostal)
		assert.Contains(t, invoice.Sujetos.Destinatarios.IDDestinatario[0].Direccion, "PO-745")
	})

	t.Run("should contain the right id for abroad customers", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = &tax.Identity{Country: "GB", Code: "PP-123456-S"}
		goblInvoice.Customer.Name = "Abroad Co LLC"

		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)

		assert.Equal(t, "GB", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.CodigoPais)
		assert.Equal(t, "PP-123456-S", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.ID)
		assert.Equal(t, "02", invoice.Sujetos.Destinatarios.IDDestinatario[0].IDOtro.IDType)
		assert.Equal(t, "Abroad Co LLC", invoice.Sujetos.Destinatarios.IDDestinatario[0].ApellidosNombreRazonSocial)
	})

	t.Run("should not include customer if no tax ID present", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = nil
		goblInvoice.Customer.Identities = nil

		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)

		assert.Empty(t, invoice.Sujetos.Destinatarios)
	})

	t.Run("should change the document type from the default (02) if stated", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer.TaxID = nil
		goblInvoice.Customer.Identities = []*org.Identity{
			{
				Key:  "passport",
				Code: "PP123456S",
			},
		}
		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)

		dest := invoice.Sujetos.Destinatarios.IDDestinatario[0]
		assert.Equal(t, "03", dest.IDOtro.IDType)
		assert.Equal(t, "PP123456S", dest.IDOtro.ID)
		assert.Empty(t, dest.IDOtro.CodigoPais)
	})

	t.Run("should allow having no customer (useful for simplied invoices)", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.Customer = nil

		invoice, _ := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)

		assert.Nil(t, invoice.Sujetos.Destinatarios)
	})

	t.Run("should reject simplified invoice with customer tax ID", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.SetTags(tax.TagSimplified)

		_, err := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "customer tax ID must not be set for simplified invoices")
	})

	t.Run("should allow simplified invoice with customer without tax ID", func(t *testing.T) {
		goblInvoice := test.LoadInvoice("sample-invoice.json")
		goblInvoice.SetTags(tax.TagSimplified)
		goblInvoice.Customer.TaxID = nil

		invoice, err := convert.NewTicketBAI(goblInvoice, ts, role, convert.ZoneBI)
		require.NoError(t, err)
		assert.Nil(t, invoice.Sujetos.Destinatarios)
		assert.Equal(t, "S", invoice.Factura.CabeceraFactura.FacturaSimplificada)
	})

	t.Run("fail when charges are present since they aren't supported", func(t *testing.T) {
		inv := test.LoadInvoice("sample-invoice.json")
		inv.Lines[0].Charges = []*bill.LineCharge{{Amount: num.MakeAmount(100, 2)}}

		_, err := convert.NewTicketBAI(inv, ts, role, convert.ZoneBI)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "charges are not supported")

		inv.Lines[0].Charges = nil
		inv.Charges = []*bill.Charge{{Amount: num.MakeAmount(100, 2)}}

		_, err = convert.NewTicketBAI(inv, ts, role, convert.ZoneBI)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "charges are not supported")
	})
}

func TestDocumentParsing(t *testing.T) {
	path := test.Path("test", "data", "out", "sample-invoice.xml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	in := new(convert.TicketBAI)
	err = xml.Unmarshal(data, in)
	require.NoError(t, err)

	assert.Equal(t, "1089.00", in.Factura.DatosFactura.ImporteTotalFactura)
	assert.Equal(t, "900.00", in.Factura.TipoDesglose.DesgloseFactura.Sujeta.NoExenta.DetalleNoExenta[0].DesgloseIVA.DetalleIVA[0].BaseImponible)
	assert.Equal(t, "AQAB", in.Signature.KeyInfo.KeyValue.RSA.Exponent)
}
