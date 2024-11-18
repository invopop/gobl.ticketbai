// Package ca is used to provide the root certificates used by the TicketBAI
// services. Certificates where sourced from:
//
// - https://www.izenpe.eus/descarga-de-certificados/webize01-cndoctecnica/es/
//
// And converted to PEM format:
//
//	openssl x509 -in AAPPNR_cert_sha256.crt -outform PEM -out AAPPNR_cert_sha256.pem
package ca

import "embed"

//go:embed *.pem

// Content contains the root certificates used by the TicketBAI services.
var Content embed.FS
