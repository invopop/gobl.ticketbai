package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

// LoadEnvelope will load a test JSON document as a GOBL Envelope
// from the test folder
func LoadEnvelope(name string) (*gobl.Envelope, error) {
	envelopeReader, err := os.Open(TestPath("data", name))
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(envelopeReader)
	if err != nil {
		return nil, err
	}

	envelope := new(gobl.Envelope)
	err = json.Unmarshal(buf.Bytes(), envelope)
	if err != nil {
		return nil, err
	}

	return envelope, nil
}

// LoadInvoice grabs the gobl envelope and attempts to extract the invoice payload
func LoadInvoice(name string) (*bill.Invoice, error) {
	env, err := LoadEnvelope(name)
	if err != nil {
		return nil, err
	}

	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, errors.New("envelope does not contain an invoice")
	}

	return inv, nil
}

// TestPath joins the provided elements to the project's test folder
func TestPath(elements ...string) string {
	elements = append([]string{"test"}, elements...)
	return Path(elements...)
}

// Path joins the provided elements to the project root
func Path(elements ...string) string {
	np := []string{RootPath()}
	np = append(np, elements...)
	return path.Join(np...)
}

// RootPath returns the project root, regardless of current working directory.
func RootPath() string {
	cwd, _ := os.Getwd()

	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}

	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}

	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}
