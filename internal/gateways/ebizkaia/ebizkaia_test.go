package ebizkaia

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"testing"
)

func gunzip(t *testing.T, data []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	return out
}

func TestNewCreateRequestModelo140(t *testing.T) {
	sup := &Supplier{
		Year:     "2026",
		NIF:      "12345678Z",
		Name:     "Individual Bizkaia",
		Model:    Modelo140,
		Activity: "722300",
	}
	payload := []byte("<TicketBai>fake</TicketBai>")

	req, err := NewCreateRequest(sup, payload)
	if err != nil {
		t.Fatalf("NewCreateRequest: %v", err)
	}

	body := gunzip(t, req.Payload)

	checks := []string{
		`lrpficfcsgap:LROEPF140IngresosConFacturaConSGAltaPeticion`,
		`xmlns:lrpficfcsgap="https://www.batuz.eus/fitxategiak/batuz/LROE/esquemas/LROE_PF_140_1_1_Ingresos_ConfacturaConSG_AltaPeticion_V1_0_2.xsd"`,
		`<Modelo>140</Modelo>`,
		`<Capitulo>1</Capitulo>`,
		`<Subcapitulo>1.1</Subcapitulo>`,
		`<Ingresos>`,
		`<Ingreso>`,
		`<TicketBai>PFRpY2tldEJhaT5mYWtlPC9UaWNrZXRCYWk+</TicketBai>`,
		`<Renta>`,
		`<DetalleRenta>`,
		`<Epigrafe>722300</Epigrafe>`,
	}
	for _, want := range checks {
		if !bytes.Contains(body, []byte(want)) {
			t.Errorf("payload missing %q\npayload:\n%s", want, body)
		}
	}

	// Make sure the 240 wrapper does NOT leak in.
	if bytes.Contains(body, []byte("FacturasEmitidas")) {
		t.Errorf("payload should not contain FacturasEmitidas wrapper:\n%s", body)
	}

	var jhead N3Header
	if err := json.Unmarshal(req.Header, &jhead); err != nil {
		t.Fatalf("json header: %v", err)
	}
	if jhead.DatosRelevantes.Modelo != Modelo140 {
		t.Errorf("N3 header mode = %q, want %s", jhead.DatosRelevantes.Modelo, Modelo140)
	}
}

func TestNewCreateRequestModelo240(t *testing.T) {
	for _, model := range []string{Modelo240, ""} {
		t.Run("model="+model, func(t *testing.T) {
			sup := &Supplier{
				Year:  "2026",
				NIF:   "B64847106",
				Name:  "Some Co SL",
				Model: model,
			}
			req, err := NewCreateRequest(sup, []byte("<TicketBai>fake</TicketBai>"))
			if err != nil {
				t.Fatalf("NewCreateRequest: %v", err)
			}

			body := gunzip(t, req.Payload)

			checks := []string{
				`lrpjfecsgap:LROEPJ240FacturasEmitidasConSGAltaPeticion`,
				`<Modelo>240</Modelo>`,
				`<FacturasEmitidas>`,
				`<FacturaEmitida>`,
			}
			for _, want := range checks {
				if !bytes.Contains(body, []byte(want)) {
					t.Errorf("payload missing %q\npayload:\n%s", want, body)
				}
			}
			if bytes.Contains(body, []byte("Ingresos")) {
				t.Errorf("payload should not contain Ingresos wrapper:\n%s", body)
			}
			if bytes.Contains(body, []byte("Renta")) {
				t.Errorf("payload should not contain Renta block:\n%s", body)
			}

			var jhead N3Header
			if err := json.Unmarshal(req.Header, &jhead); err != nil {
				t.Fatalf("json header: %v", err)
			}
			if jhead.DatosRelevantes.Modelo != Modelo240 {
				t.Errorf("N3 header mode = %q, want %s", jhead.DatosRelevantes.Modelo, Modelo240)
			}
		})
	}
}

func TestNewCancelRequestModelo140(t *testing.T) {
	sup := &Supplier{
		Year:  "2026",
		NIF:   "12345678Z",
		Name:  "Individual Bizkaia",
		Model: Modelo140,
	}
	req, err := NewCancelRequest(sup, []byte("<AnulaTicketBai/>"))
	if err != nil {
		t.Fatalf("NewCancelRequest: %v", err)
	}

	body := gunzip(t, req.Payload)

	checks := []string{
		`lrpficfcsgap:LROEPF140IngresosConFacturaConSGAnulacionPeticion`,
		`<Modelo>140</Modelo>`,
		`<Ingresos>`,
		`<Ingreso>`,
		`<AnulacionTicketBai>`,
	}
	for _, want := range checks {
		if !bytes.Contains(body, []byte(want)) {
			t.Errorf("payload missing %q\npayload:\n%s", want, body)
		}
	}
	if bytes.Contains(body, []byte("<Renta>")) {
		t.Errorf("cancel payload must not contain <Renta> block:\n%s", body)
	}
}

func TestNewCreateRequestModelo140EmptyActivity(t *testing.T) {
	// Empty Activity is accepted by the factory; the converter is responsible for
	// guaranteeing presence. Batuz will reject the payload server-side if Epigrafe
	// is missing, surfacing a clear validation error.
	sup := &Supplier{
		Year:  "2026",
		NIF:   "12345678Z",
		Name:  "Individual Bizkaia",
		Model: Modelo140,
	}
	req, err := NewCreateRequest(sup, []byte("<TicketBai>fake</TicketBai>"))
	if err != nil {
		t.Fatalf("NewCreateRequest with empty Activity should not error: %v", err)
	}
	body := gunzip(t, req.Payload)
	if !bytes.Contains(body, []byte(`<DetalleRenta>`)) {
		t.Errorf("payload missing <DetalleRenta>:\n%s", body)
	}
	if !bytes.Contains(body, []byte(`<Epigrafe></Epigrafe>`)) {
		t.Errorf("expected empty <Epigrafe></Epigrafe> when Activity is empty:\n%s", body)
	}
}

func TestLROEPF140AltaRespuestaFirstError(t *testing.T) {
	r := &LROEPF140IngresosConFacturaConSGAltaRespuesta{}
	if got := r.FirstErrorCode(); got != "" {
		t.Errorf("FirstErrorCode on empty response = %q, want empty", got)
	}
	if got := r.FirstErrorDescription(); got != "" {
		t.Errorf("FirstErrorDescription on empty response = %q, want empty", got)
	}

	r.Registros = &RegistrosFacturaConSGType{
		Registro: []*RegistroFacturaConSGType{
			{
				SituacionRegistro: &SituacionRegistroType{
					CodigoErrorRegistro:        "B4_2000003",
					DescripcionErrorRegistroES: "Algo falló",
				},
			},
		},
	}
	if got := r.FirstErrorCode(); got != "B4_2000003" {
		t.Errorf("FirstErrorCode = %q, want B4_2000003", got)
	}
	if got := r.FirstErrorDescription(); got != "Algo falló" {
		t.Errorf("FirstErrorDescription = %q, want %q", got, "Algo falló")
	}
}
