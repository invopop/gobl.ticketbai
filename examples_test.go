package ticketbai_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/gobl.ticketbai/doc"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/xmldsig"
	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	msgMissingOutFile    = "output file %s missing, run tests with `--update` flag to create"
	msgUnmatchingOutFile = "output file %s does not match, run tests with `--update` flag to update"
)

func TestXMLGeneration(t *testing.T) {
	schema, err := loadSchema()
	require.NoError(t, err)

	examples, err := lookupExamples()
	require.NoError(t, err)

	tbai, err := loadTBAIClient()
	require.NoError(t, err)

	for _, example := range examples {
		name := fmt.Sprintf("should convert %s example file successfully", example)

		t.Run(name, func(t *testing.T) {
			data, err := convertExample(tbai, example)
			require.NoError(t, err)

			outPath := test.Path("test", "data", "out",
				strings.TrimSuffix(example, ".json")+".xml",
			)

			if *test.UpdateOut {
				errs := validateDoc(schema, data)
				for _, e := range errs {
					assert.NoError(t, e)
				}
				if len(errs) > 0 {
					assert.Fail(t, "Invalid XML:\n"+string(data))
					return
				}

				err = os.WriteFile(outPath, data, 0644)
				require.NoError(t, err)

				return
			}

			expected, err := os.ReadFile(outPath)

			require.False(t, os.IsNotExist(err), msgMissingOutFile, filepath.Base(outPath))
			require.NoError(t, err)
			require.Equal(t, string(expected), string(data), msgUnmatchingOutFile, filepath.Base(outPath))
		})
	}
}

func loadSchema() (*xsd.Schema, error) {
	schemaPath := test.Path("test", "schema", "schema.xsd")
	schema, err := xsd.ParseFromFile(schemaPath)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func loadTBAIClient() (*ticketbai.Client, error) {
	pass, err := os.ReadFile(
		test.Path("test", "certs", "EntitateOrdezkaria_RepresentanteDeEntidad_pin.txt"),
	)
	if err != nil {
		return nil, err
	}

	cert, err := xmldsig.LoadCertificate(
		test.Path("test", "certs", "EntitateOrdezkaria_RepresentanteDeEntidad.p12"),
		string(pass),
	)
	if err != nil {
		return nil, err
	}

	ts, err := time.Parse(time.RFC3339, "2022-02-01T04:00:00Z")
	if err != nil {
		return nil, err
	}

	return ticketbai.New(&ticketbai.Software{
		License: "My License",
		NIF:     "12345678A",
		Name:    "My Software",
		Version: "1.0",
	},
		ticketbai.ZoneBI,
		ticketbai.WithCertificate(cert),
		ticketbai.WithCurrentTime(ts),
		ticketbai.WithThirdPartyIssuer(),
	)
}

func lookupExamples() ([]string, error) {
	examples, err := filepath.Glob(test.Path("test", "data", "*.json"))
	if err != nil {
		return nil, err
	}

	for i, example := range examples {
		examples[i] = filepath.Base(example)
	}

	return examples, nil
}

func convertExample(tc *ticketbai.Client, example string) ([]byte, error) {
	env := test.LoadEnvelope(example)

	td, err := tc.Convert(env)
	if err != nil {
		return nil, err
	}

	err = tc.Fingerprint(td, &doc.ChainData{
		Series:    "AF",
		Code:      "1234567890",
		IssueDate: "01-01-2021",
		Signature: strings.Repeat("1234567890", 20),
	})
	if err != nil {
		return nil, err
	}

	err = tc.Sign(td, env)
	if err != nil {
		return nil, err
	}

	return td.BytesIndent()
}

func validateDoc(schema *xsd.Schema, doc []byte) []error {
	xmlDoc, err := libxml2.ParseString(string(doc))
	if err != nil {
		return []error{err}
	}

	err = schema.Validate(xmlDoc)
	if err != nil {
		return err.(xsd.SchemaValidationError).Errors()
	}

	return nil
}
