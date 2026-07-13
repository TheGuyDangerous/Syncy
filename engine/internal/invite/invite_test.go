package invite

import (
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	want := Code{
		DeviceID:  "MFRGGZDFMZTWQ2LK",
		Name:      "workstation",
		Endpoints: []string{"192.168.1.10:22067", "[2001:db8::1]:22067", "203.0.113.4:41000"},
	}
	got, err := Decode(Encode(want))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestRoundTripMinimal(t *testing.T) {
	want := Code{DeviceID: "ABCDEF"}
	got, err := Decode(Encode(want))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestDecodeTrimsWhitespace(t *testing.T) {
	token := "  " + Encode(Code{DeviceID: "X"}) + "\n"
	if _, err := Decode(token); err != nil {
		t.Errorf("Decode with surrounding whitespace: %v", err)
	}
}

func TestEncodeIsCopyPasteable(t *testing.T) {
	token := Encode(Code{DeviceID: "ID", Endpoints: []string{"10.0.0.2:22067"}})
	if !strings.HasPrefix(token, Prefix) {
		t.Errorf("token %q should start with %q", token, Prefix)
	}
	if strings.ContainsAny(token, " \t\r\n+/=") {
		t.Errorf("token %q contains characters unsafe for copy-paste", token)
	}
}

func TestDecodeMalformed(t *testing.T) {
	badJSON := Prefix + base64.RawURLEncoding.EncodeToString([]byte("{not json"))
	noID := Prefix + base64.RawURLEncoding.EncodeToString([]byte(`{"name":"x"}`))
	badEndpoint := Encode(Code{DeviceID: "X", Endpoints: []string{"no-port"}})

	cases := map[string]string{
		"empty":          "",
		"no prefix":      "asdf",
		"wrong prefix":   "SYNCY9-abcd",
		"bad base64":     Prefix + "!!!!",
		"bad json":       badJSON,
		"missing id":     noID,
		"bad endpoint":   badEndpoint,
		"prefix only":    Prefix,
		"spaces in body": Prefix + "ab cd",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(token); err == nil {
				t.Errorf("Decode(%q) should have failed", token)
			}
		})
	}
}

func TestDecodeErrorKinds(t *testing.T) {
	if _, err := Decode("nope"); !errors.Is(err, ErrNotInvite) {
		t.Errorf("error = %v, want ErrNotInvite", err)
	}
	noID := Prefix + base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	if _, err := Decode(noID); !errors.Is(err, ErrMissingID) {
		t.Errorf("error = %v, want ErrMissingID", err)
	}
}
