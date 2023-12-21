package doc_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl.ticketbai/internal/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignatureGeneration(t *testing.T) {
	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	require.NoError(t, err)

	cert, err := xmldsig.LoadCertificate(test.Path("test", "certs", "EntitateOrdezkaria_RepresentanteDeEntidad.p12"), "IZDesa2021")
	require.NoError(t, err)

	t.Run("should set the proper signer role", func(t *testing.T) {
		role := doc.IssuerRoleCustomer

		goblInvoice, _ := test.LoadInvoice("sample-invoice.json")
		invoice, _ := doc.NewTicketBAI(goblInvoice, ts, role)

		err := invoice.Sign("TEST", cert)
		require.NoError(t, err)

		assert.Equal(t,
			string(doc.XAdESCustomer),
			invoice.Signature.Object.QualifyingProperties.SignedProperties.
				SignatureProperties.SignerRole.ClaimedRoles.ClaimedRole[0],
		)
	})

}
