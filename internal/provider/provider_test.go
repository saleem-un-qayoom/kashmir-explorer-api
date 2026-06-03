package provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNz(t *testing.T) {
	if nz("") != nil {
		t.Error("nz(\"\") should be nil")
	}
	if nz("   ") != nil {
		t.Error("nz(blank) should be nil")
	}
	if got := nz("Srinagar"); got == nil || *got != "Srinagar" {
		t.Errorf("nz(\"Srinagar\") = %v, want pointer to Srinagar", got)
	}
}

func decodeBody(t *testing.T, body string) (providerInput, error) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	var in providerInput
	err := in.decode(r)
	return in, err
}

func TestProviderInputDecodeDefaults(t *testing.T) {
	in, err := decodeBody(t, `{"name":"  Dal Houseboat  "}`)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if in.Name != "Dal Houseboat" {
		t.Errorf("Name = %q, want trimmed \"Dal Houseboat\"", in.Name)
	}
	if in.Type != "guide" {
		t.Errorf("Type = %q, want default \"guide\"", in.Type)
	}
	if in.PriceUnit != "per-person" {
		t.Errorf("PriceUnit = %q, want default \"per-person\"", in.PriceUnit)
	}
}

func TestProviderInputDecodeKeepsExplicitValues(t *testing.T) {
	in, err := decodeBody(t, `{"name":"Heli Co","type":"heli","price_unit":"per-trip","price_inr":42000}`)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if in.Type != "heli" || in.PriceUnit != "per-trip" || in.PriceINR != 42000 {
		t.Errorf("got %+v, want explicit heli/per-trip/42000", in)
	}
}

func TestProviderInputDecodeInvalidJSON(t *testing.T) {
	if _, err := decodeBody(t, `{not json`); err == nil {
		t.Fatal("expected error decoding invalid JSON")
	}
}
