# GOBL to TicketBAI

Go library to convert [GOBL](https://github.com/invopop/gobl) invoices into TicketBAI declarations and send them to the Basque Country web services.

This library assumes that clients will handle a local database of previous invoices in order to comply with the local requirements of chaining all invoices together.

Copyright [Invopop Ltd.](https://invopop.com) 2023. Released publicly under the [GNU Affero General Public License v3.0](LICENSE). For commercial licenses please contact the [dev team at invopop](mailto:dev@invopop.com). For contributions to this library to be accepted, we will require you to accept transferring your copyright to Invopop Ltd.

## Source

The basque country is split into 4 "foral" tax agencies. 3 of those tax agencies decided to adopt the TicketBAI format. Each of the three agencies has their own set of documentation and integration definitions.

Links to key information for each agency are described in the following subchapters.

### Bizkaia

- BOB: https://www.batuz.eus/fitxategiak/batuz/normativa/2020%2009%2011%20ORDEN%20FORAL%201482-2020,%20de%209%20de%20septiembre.pdf?hash=929e00a8bbd774ee911841d36e3301e2
- XSD basic TicketBAI request: https://www.batuz.eus/fitxategiak/batuz/ticketbai/ticketBaiV1-2.xsd
- XSD complete list: https://www.batuz.eus/fitxategiak/Batuz/LROE/esquemas/Esquemas%20XSD.7z
- Ley IVA Foral, con datos útiles sobre exclusiones (a partir de página 24): https://www.bizkaia.eus/ogasuna/zerga_arautegia/indarreko_arautegia/pdf/ca_7_1994.pdf
- FAQ Gipuzkoa: https://www.gipuzkoa.eus/es/web/ogasuna/ticketbai/faq (download the PDF from that page, it's much easier to read.)

## Usage

### Go Package

You must have first created a GOBL Envelope containing an Invoice that you'd like to send to one of the TicketBAI web services.

For the document to accepted, the supplier contained in the invoice should have a "Tax ID" that includes:

- A country code set to `ES`
- A zone code set to region of one of the three Basque Country tax agencies, i.e. `BI`, `SS`, or `VI`. (We don't consider the address field reliable for this.)

The following is an example of how the GOBL TicketBAI package could be used:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/gobl"
	ticketbai "github.com/invopop/gobl.ticketbai"
	"github.com/invopop/xmldsig"
)

func main() {
	// Load sample envelope:
	data, _ := os.ReadFile("./test/data/sample-invoice.json")

	env := new(gobl.Envelope)
	if err := json.Unmarshal(data, env); err != nil {
		panic(err)
	}

	// Prepare software configuration:
	soft := &ticketbai.Software{
		License: "XYZ",        // provided by tax agency
		NIF:     "B123456789", // Software company's tax code
		Name:    "Invopop",    // Name of application
		Version: "v0.1.0",     // Software version
	}

	// Load sample certificate:
	cert, err := xmldsig.LoadCertificate(
		"./test/certs/EnpresaZigilua_SelloDeEmpresa.p12", "IZDesa2021")
	if err != nil {
		panic(err)
	}

	// Instantiate the TicketBAI client:
	tbai, err := ticketbai.New(soft,
		ticketbai.WithCertificate(cert), // Use the certificate previously loaded
		ticketbai.WithSupplierIssuer(),  // The issuer is the invoice's supplier
		ticketbai.InTesting(),           // Use the tax agency testing environment
	)
	if err != nil {
		panic(err)
	}

	// Create a new TBAI document:
	doc, err := tbai.NewDocument(env)
	if err != nil {
		panic(err)
	}

	// Create the document fingerprint:
	if err = tbai.Fingerprint(doc, prev); err != nil {
		panic(err)
	}

	// Sign the document:
	if err := tbai.Sign(doc); err != nil {
		panic(err)
	}

	// Create the XML output
	bytes, err := doc.BytesIndent()
	if err != nil {
		panic(err)
	}

	// Do something with the output
	fmt.Println("Document created:\n", string(bytes))
}
```

## Command Line

The GOBL TicketBAI package tool also includes a command line helper. You can find pre-built [gobl.cfdi binaries](https://github.com/invopop/gobl.ticketbai/releases) in the github repository, or install manually in your Go environment with:

```bash
go install github.com/invopop/gobl.ticketbai
```

Usage is very straightforward:

```bash
gobl.ticketbai convert ./test/data/invoice.json
```

At the moment, it's not possible to add a fingertip or sign TicketBAI files using the CLI.

## Limitations

- Tickebai allows more than one customer per invoice, but GOBL only has one possible customer.

- Invoices should have a note of type general that will be used as a general description of the invoice. If an invoice is missing this info, it will be rejected with an error.

- Currently GOBL does not allow to distinguish between different VAT regimes. Ticketbai format requires a list of the different regimes applied to the invoice so currently only equivalence surcharge and general regime are available (for a complete list of the other possibilities you can check the documentation on https://www.batuz.eus/es/documentacion-tecnica)

## Tags, Keys and Extensions

In order to provide the supplier specific data required by TicketBAI, invoices need to include a bit of extra data. We've managed to simplify these into specific cases.

### Tax Tags

Invoice tax tags can be added to invoice documents in order to reflect a special situation. The following schemes are supported:

- `simplified-scheme` - a retailer operating under a simplified tax regime (regimen simplificado) that must indicate that all of their sales are under this scheme. This implies that all operations in the invoice will have the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag set to `S`.
- `reverse-charge` - B2B services or goods sold to a tax registered EU member who will pay VAT on the suppliers behalf. Implies that all items will be classified under the `TipoNoExenta` value of `S2`.
- `customer-rates` - B2C services, specifically for the EU digital goods act (2015) which imply local taxes will be applied. All items will specify the `DetalleNoSujeta` cause of `RL`.

## Tax Extensions

The following extension can be applied to each line tax:

- `es-tbai-product` – allows to correctly group the invoice's lines taxes in the TicketBAI breakdowns (a.k.a. desgloses). These are the valid values:
	- `services` - indicates that the product being sold is a service (as opposed to a physical good). Services are accounted in the `DesgloseTipoOperacion > PrestacionServicios` breakdown of invoices to foreign customers. By default, all items are considered services.
	- `goods` - indicates that the product being sold is a physical good. Products are accounted in the `DesgloseTipoOperacion > Entrega` breakdown of invoices to foreign customers.
	- `resale` - indicates that a line item is sold without modification from a provider under the Equalisation Charge scheme. (This implies that the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag will be set to `S`).

- `es-tbai-exemption` - identifies the specific TicketBAI reason code as to why taxes should not be applied to the line according to the whole set of exemptions or not-subject scenarios defined in the law. It has to be set along with the tax rate value of `exempt`. These are the valid values:
	- `E1` – Exenta por el artículo 20 de la Norma Foral del IVA
  - `E2` – Exenta por el artículo 21 de la Norma Foral del IVA
  - `E3` – Exenta por el artículo 22 de la Norma Foral del IVA
  - `E4` – Exenta por el artículo 23 y 24 de la Norma Foral del IVA
  - `E5` – Exenta por el artículo 25 de la Norma Foral del IVA
  - `E6` – Exenta por otra causa
  - `OT` – No sujeto por el artículo 7 de la Norma Foral de IVA / Otros supuestos
  - `RL` – No sujeto por reglas de localización (*)

_(*) As noted elsewhere, `RL` will be set automatically set in invoices using the `customer-rates` tax tag. It can also be set explicitly using the `es-tbai-exemption` extension in invoices not using that tag._

### Use-Cases

Under what situations should the TicketBAI system be expected to function:

- B2B & B2C: regular national invoice with VAT. Operation with minimal data.
- B2B Provider to Retailer: Include equalisation surcharge VAT rates
- B2B Retailer: Same as regular invoice, except with invoice lines that include `ext[es-tbai-product] = resale` when the goods being provided are being sold without modification (recargo de equivalencia), very much related to the next point.
- B2B Retailer Simplified: Include the simplified scheme key. (This implies that the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag will be set to `S`).
- EU B2B: Reverse charge EU export, scheme: reverse-charge taxes calculated, but not applied to totals. By default all line items assumed to be services. Individual lines can use the `ext[es-tbai-product] = goods` value to identify when the line is a physical good. Operations like this are normally assigned the TipoNoExenta value of S2. If however the service or goods are exempt of tax, each line's tax `ext[exempt]` field can be used to identify a reason.
- EU B2C Digital Goods: use tax tag `customer-rates`, that applies VAT according to customer location. In TicketBAI, these cases are "not subject" to tax, and thus should have the cause RL (por reglas de localización).

## Test Data

Some sample test data is available in the `./test` directory.

If you make any modifications to the source YAML files, the JSON envelopes will need to be updated.

Make sure you have the GOBL CLI installed ([more details](https://docs.gobl.org/quick-start/cli)).
