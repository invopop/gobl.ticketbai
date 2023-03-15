package ebizkaia

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl.ticketbai/internal/doc"
)

// Bizkaia has extra complications when sending documents, so we define all the additional
// XML wrappers here.

// Constants used in headers
const (
	concepto           = "LROE"
	apartado1          = "1"
	apartado1_1        = "1.1"
	modelo240          = "240"
	schemaLROE240ConSG = "https://www.batuz.eus/fitxategiak/batuz/LROE/esquemas/LROE_PJ_240_1_1_FacturasEmitidas_ConSG_AltaPeticion_V1_0_2.xsd"
)

const (
	operacionEnumAlta                           = "A00"
	operacionEnumAltaDevolucionViajeros         = "A01"
	operacionEnumModificacion                   = "M00"
	operacionEnumModificacionDevolucionViajeros = "M01"
	operacionEnumAnulacion                      = "AN0"
	operacionEnumConsulta                       = "C00"
)

// CreateRequest tries to keep all the request information in one place ready to send to the
// Bizkaia servers. They really out did themselves while defining these connections making it
// increadibly complex.
type CreateRequest struct {
	Header  []byte // JSON special data header
	Payload []byte // already gzipped
}

// N3Header is the structure that needs to be included in requests containing
// details about what is being sent. For some reason, they decided instead of defining
// different URLs according to the use-case, to just have a single end point with a JSON
// object included in the HTTP headers. Bizarre.
type N3Header struct {
	Concepto        string            `json:"con"` // always "LROE"
	Apartado        string            `json:"apa"`
	Interesado      N3Interesado      `json:"inte"`
	DatosRelevantes N3DatosRelevantes `json:"drs"`
}

// N3Interesado defines fields that describe the taxable entity.
type N3Interesado struct {
	NIF       string `json:"nif"`
	Nombre    string `json:"nrs"`
	Apellido1 string `json:"ap1,omitempty"`
	Apellido2 string `json:"ap2,omitempty"`
}

// N3DatosRelevantes defines the struct used to store additional
// information about the request, used in the header.
type N3DatosRelevantes struct {
	Modelo    string `json:"mode"`
	Ejercicio string `json:"ejer"` // invoice issue year
}

// LROEPJ240FacturasEmitidasConSGAltaPeticion is used for uploading invoices. "240" refers to
// the type of entity which in this case is a "business", and "ConSG" means the request is
// from a "software garante", i.e. Invopop.
type LROEPJ240FacturasEmitidasConSGAltaPeticion struct {
	// XMLName xml.Name `xml:"LROEPJ240FacturasEmitidasConSGAltaPeticion"`
	XMLName       xml.Name `xml:"lrpjfecsgap:LROEPJ240FacturasEmitidasConSGAltaPeticion"`
	LROENamespace string   `xml:"xmlns:lrpjfecsgap,attr"`

	Cabecera         *Cabecera240Type
	FacturasEmitidas *FacturasEmitidasConSGCodificadoType
}

// Cabecera250Type contains the operation headers
type Cabecera240Type struct {
	Modelo             string
	Capitulo           string
	Subcapitulo        string `xml:",omitempty"`
	Operacion          string
	Version            string
	Ejercicio          string
	ObligadoTributario *NIFPersonaType
}

// FacturasEmitidasConSGCodificadoType holds an array of invoices
// to send.
type FacturasEmitidasConSGCodificadoType struct {
	FacturaEmitida []*DetalleEmitidaConSGCodificadoType // max length 1000
}

// FacturaEmitida contains the invoice to upload
type DetalleEmitidaConSGCodificadoType struct {
	TicketBai string // base64 data
}

// NIFPersonaType
type NIFPersonaType struct {
	NIF                        string
	ApellidosNombreRazonSocial string // max 120
}

// Supplier contains the details of the supplier who is making the
// request.
type Supplier struct {
	Year int    // invoice issue year
	NIF  string // Tax code
	Name string // Name of the company
}

// NewCreateRequest simplifies the process of creating a new request for a regular
// company using Invopop.
func NewCreateRequest(sup *Supplier, tbai *doc.TicketBAI) (*CreateRequest, error) {
	req := new(CreateRequest)

	head := newCabecera240Type(sup)
	facs, err := newFacturasEmitidas(tbai)
	if err != nil {
		return nil, err
	}
	body := &LROEPJ240FacturasEmitidasConSGAltaPeticion{
		LROENamespace:    schemaLROE240ConSG,
		Cabecera:         head,
		FacturasEmitidas: facs,
	}

	bdata, err := xml.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode body to XML: %w", err)
	}
	fmt.Printf("********\n%v\n******\n", string(bdata))
	req.Payload, err = compressBody(bdata)
	if err != nil {
		return nil, fmt.Errorf("compressing body: %w", err)
	}

	jhead := newN3Header(sup)
	req.Header, err = json.Marshal(jhead)
	if err != nil {
		return nil, fmt.Errorf("json header: %w", err)
	}

	// Holy fuck that was complicated.
	return req, nil
}

// newFacturasEmitidas will encode the invoice data to base64 and instantiate a
// new object to include in the XML message body.
func newFacturasEmitidas(tbais ...*doc.TicketBAI) (*FacturasEmitidasConSGCodificadoType, error) {
	b := &FacturasEmitidasConSGCodificadoType{
		FacturaEmitida: make([]*DetalleEmitidaConSGCodificadoType, len(tbais)),
	}
	for i, tbai := range tbais {
		d, err := tbai.Bytes()
		if err != nil {
			return nil, fmt.Errorf("tbai %d: failed to encode to bytes: %w", i, err)
		}
		b.FacturaEmitida[i] = &DetalleEmitidaConSGCodificadoType{
			TicketBai: base64.StdEncoding.EncodeToString(d),
		}
	}
	return b, nil
}

func newCabecera240Type(sup *Supplier) *Cabecera240Type {
	head := &Cabecera240Type{
		Modelo:      modelo240,
		Capitulo:    apartado1,
		Subcapitulo: apartado1_1,
		Operacion:   operacionEnumAlta,
		Version:     "1.0",
		Ejercicio:   fmt.Sprintf("%d", sup.Year),
		ObligadoTributario: &NIFPersonaType{
			NIF:                        sup.NIF,
			ApellidosNombreRazonSocial: sup.Name,
		},
	}
	return head
}

func newN3Header(sup *Supplier) *N3Header {
	// prepare the request data header
	head := &N3Header{
		Concepto: concepto,
		Apartado: apartado1_1, // for invoices
		Interesado: N3Interesado{
			NIF:    sup.NIF,
			Nombre: sup.Name,
		},
		DatosRelevantes: N3DatosRelevantes{
			Modelo:    modelo240,
			Ejercicio: fmt.Sprintf("%d", sup.Year),
		},
	}
	return head
}

func compressBody(data []byte) ([]byte, error) {
	// Gzip the data
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(data); err != nil {
		return nil, fmt.Errorf("compressing: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
