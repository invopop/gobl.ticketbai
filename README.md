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
    "github.com/invopop/gobl.ticketbai"
)

func main {
	// Prepare software configuration
	soft := &ticketbai.Software{
		License: "XYZ",    // provided by tax agency
		NIF: "B123456789", // Software company's tax code
		Name: "Invopop",   // Name of application
		Version: "v0.1.0", // Software version
	}

	// Instantiate a the TicketBAI client:

	// TODO!
}
```

### CLI

If you cannot use the Golang packages directly, the CLI tool can be an easy way to generate and send XML documents to the TicketBAI services.

## Special Schemes and Line Meta

In order to provide the supplier specific data required by TicketBAI, invoices need to include a bit of extra data. We've managed to simplify these into specific cases.

### Schemes

Schemes can be added to invoice documents in order to reflect a special situation. The following schemes are supported:

- `simplified` - a retailer operating under a simplified tax regime (regimen simplificado) that must indicate that all of their sales are under this scheme. This implies that all operations in the invoice will have the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag set to `Y`.
- `reverse-charge` - B2B services or goods sold to a tax registered EU member who will pay VAT on the suppliers behalf. Implies that all items will be classified under the `TipoNoExenta` value of `S2`.
- `customer-rates` - B2C services, specifically for the EU digital goods act (2015) which imply local taxes will be applied.

### Invoice Line Tags

Some lines may require additional "tags" to correctly group them in the final TicketBAI report. The currently supported invoice line tags are:

- `provider` - indicates that a line item is sold without modification from a provider under the Equalisation Charge scheme. (This implies that the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag will be set to Y).
- `services` -
- `goods` -
- `exempt` - identifies the specific reason as to why taxes should not be applied to the line according to the whole set of exemptions defined in the law. This tag cannot be used alone, and must be presented as one of the following:
  - `exempt+article-20` - (`E1`) Exenta por el artículo 20 de la Norma Foral del IVA
  - `exempt+article-21` - (`E2`) Exenta por el artículo 21 de la Norma Foral del IVA
  - `exempt+article-22` - (`E3`) Exenta por el artículo 22 de la Norma Foral del IVA
  - `exempt+article-23` - (`E4`) Exenta por el artículo 23 y 24 de la Norma Foral del IVA
  - `exempt+article-25` - (`E5`) Exenta por el artículo 25 de la Norma Foral del IVA
  - `exempt+other` - (`E6`) Exenta por otra causa

\***\* TO DELETE \*\***

The following keys may be set inside the `meta` property of each line:

- `source` - when "`provider`" indicates that a line item is sold without modification from a provider under the Equalisation Charge scheme. (This implies that the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag will be set to Y).
- `product` - when set to "`goods`", indicates that the product being sold is a physical good. By default we assume services are being sold.
- `exempt` - two-letter-code - identifies the specific reason as to why taxes should not be applied to the line according to the whole set of exemptions defined in the law. The main codes that a user can provide are:
  - `E1` - Exenta por el artículo 20 de la Norma Foral del IVA
  - `E2` - Exenta por el artículo 21 de la Norma Foral del IVA
  - `E3` - Exenta por el artículo 22 de la Norma Foral del IVA
  - `E4` - Exenta por el artículo 23 y 24 de la Norma Foral del IVA
  - `E5` - Exenta por el artículo 25 de la Norma Foral del IVA
  - `E6` - Exenta por otra causa

\***\* END DELETE \*\***

### Use-Cases

Under what situations should the TicketBAI system be expected to function:

- B2B & B2C: regular national invoice with VAT. Operation with minimal data.
- B2B Provider to Retailer: Include equalisation surcharge VAT rates
- B2B Retailer: Same as regular invoice, except with invoice lines that include `meta[source] = provider` when the goods being provided are being sold without modification (recargo de equivalencia), very much related to the next point.
- B2B Retailer Simplified: Include the simplified scheme key. (This implies that the `OperacionEnRecargoDeEquivalenciaORegimenSimplificado` tag will be set to Y).
- EU B2B: Reverse charge EU export, scheme: reverse-charge taxes calculated, but not applied to totals. By default all line items assumed to be services. Individual rows can use the `meta[product] = goods` value to identify when the line is a physical good. Operations like this are normally assigned the TipoNoExenta value of S2. If however the service or goods are exempt of tax, each line's `meta[exempt]` field can be used to identify a reason.
- EU B2C Digital Goods: use scheme `customer-rates`, that applies VAT according to customer location. In TicketBAI, these cases are "not subject" to tax, and thus should have the cause RL (por reglas de localización).

## Test Data

Some sample test data is available in the `./test` directory.

If you make any modifications to the source YAML files, the JSON envelopes will need to be updated.

Make sure you have the GOBL CLI installed ([more details](https://docs.gobl.org/quick-start/cli)):

```
go install github.com/invopop/gobl.cli/cmd/gobl@latest
gobl keygen
```

Then sign the documents:

```
cd test
gobl sign --indent sample-invoice.yaml > sample-invoice.json
```
