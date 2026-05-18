package ticketbai_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/gobl.ticketbai/convert"
	"github.com/invopop/gobl.ticketbai/internal/gateways"
	"github.com/invopop/gobl.ticketbai/test"
	"github.com/invopop/xmldsig"
	"github.com/stretchr/testify/require"
)

const (
	msgMissingOutFile    = "output file %s missing, run tests with `--update` flag to create"
	msgUnmatchingOutFile = "output file %s does not match, run tests with `--update` flag to update"
)

func TestXMLGeneration(t *testing.T) {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not installed; install libxml2 to enable XSD validation")
	}

	schemaPath := test.Path("test", "schema", "schema.xsd")
	catalogPath := test.Path("test", "schema", "catalog.xml")

	examples, err := lookupExamples()
	require.NoError(t, err)

	tbai, err := loadTBAIClient()
	require.NoError(t, err)

	for _, example := range examples {
		name := fmt.Sprintf("should convert %s example file successfully", example)

		t.Run(name, func(t *testing.T) {
			data, err := convertExample(tbai, example)
			require.NoError(t, err)

			// Validate against the TicketBAI XSD on every run so
			// CI catches schema regressions even without --update.
			require.NoError(t, validateDoc(xmllint, schemaPath, catalogPath, data))

			outPath := test.Path("test", "data", "out",
				strings.TrimSuffix(example, ".json")+".xml",
			)

			if *test.UpdateOut {
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
		Licenses: ticketbai.Licenses{
			gateways.EnvironmentSandbox: {
				ticketbai.ZoneBI: "My License",
			},
		},
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

	err = tc.Fingerprint(td, &convert.ChainData{
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

// validateDoc validates doc against the TicketBAI XSD by shelling out
// to xmllint. The catalog maps the W3C xmldsig schema URL to the local
// copy in test/schema/, so --nonet can stay on (newer libxml2 builds
// disable HTTP fetching by default).
func validateDoc(xmllint, schemaPath, catalogPath string, doc []byte) error {
	cmd := exec.Command(xmllint, "--noout", "--nonet", "--schema", schemaPath, "-")
	cmd.Stdin = bytes.NewReader(doc)
	cmd.Env = append(os.Environ(), "XML_CATALOG_FILES="+catalogPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xmllint: %w\n%s", err, stderr.String())
	}
	return nil
}
