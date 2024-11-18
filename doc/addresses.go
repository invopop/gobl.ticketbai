package doc

import (
	"fmt"

	"github.com/invopop/gobl/org"
)

func formatAddress(address *org.Address) string {
	if address.PostOfficeBox != "" {
		return "PO Box / Apdo " + address.PostOfficeBox
	}

	formattedAddress := fmt.Sprintf("%s, %s", address.Street, address.Number)

	if address.Block != "" {
		formattedAddress += (", " + address.Block)
	}

	if address.Floor != "" {
		formattedAddress += (", " + address.Floor)
	}

	if address.Door != "" {
		formattedAddress += (" " + address.Door)
	}

	if address.StreetExtra != "" {
		formattedAddress += ("\n" + address.StreetExtra)
	}

	return formattedAddress
}
