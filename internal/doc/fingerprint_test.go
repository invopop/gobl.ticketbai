package doc_test

import (
	"strings"
	"testing"
	"time"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFingerprintGeneration(t *testing.T) {
	var conf *doc.FingerprintConfig

	beforeEach := func(t *testing.T) *doc.TicketBAI {
		t.Helper()

		conf = &doc.FingerprintConfig{
			NIF:             "12345678A",
			SoftwareName:    "My Software",
			SoftwareVersion: "1.0",
		}

		goblInvoice, err := test.LoadInvoice("sample-invoice.json")
		require.NoError(t, err)

		ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
		require.NoError(t, err)
		role := doc.IssuerRoleThirdParty
		invoice, err := doc.NewTicketBAI(goblInvoice, ts, role, doc.ZoneBI)
		require.NoError(t, err)
		invoice.Sujetos.Emisor.NIF = test.NIF

		return invoice
	}

	t.Run("should identify the software used to create the tickebai", func(t *testing.T) {
		testCase := beforeEach(t)

		err := testCase.Fingerprint(conf)
		require.NoError(t, err)

		fingerprint := testCase.HuellaTBAI
		assert.NoError(t, err)
		assert.Equal(t, "My Software", fingerprint.Software.Nombre)
		assert.Equal(t, "1.0", fingerprint.Software.Version)
		assert.Equal(t, "12345678A", fingerprint.Software.EntidadDesarrolladora.NIF)
	})

	t.Run("should not chain invoice if no previous one from the taxpayer", func(t *testing.T) {
		testCase := beforeEach(t)
		err := testCase.Fingerprint(conf)
		require.NoError(t, err)
		fingerprint := testCase.HuellaTBAI
		assert.NoError(t, err)
		assert.Nil(t, fingerprint.EncadenamientoFacturaAnterior)
	})

	t.Run("should chain invoice with the previous one from the taxpayer", func(t *testing.T) {
		testCase := beforeEach(t)

		conf.LastSeries = "A"
		conf.LastCode = "1"
		conf.LastIssueDate = "01-01-2022"
		conf.LastSignature = strings.Repeat("1234567890", 11)

		err := testCase.Fingerprint(conf)
		require.NoError(t, err)
		fingerprint := testCase.HuellaTBAI
		chaining := fingerprint.EncadenamientoFacturaAnterior
		assert.Equal(t, "1", chaining.NumFacturaAnterior)
		assert.Equal(t, "01-01-2022", chaining.FechaExpedicionFacturaAnterior)
		assert.Equal(t, strings.Repeat("1234567890", 10), chaining.SignatureValueFirmaFacturaAnterior)
	})
}
