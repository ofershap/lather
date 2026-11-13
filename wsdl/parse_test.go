package wsdl_test

import (
	"strings"
	"testing"

	"github.com/ofershap/lather/wsdl"
)

func TestDTDRejected(t *testing.T) {
	doc := `<!DOCTYPE d [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><definitions xmlns="http://schemas.xmlsoap.org/wsdl/"/>`
	_, err := wsdl.Parse(strings.NewReader(doc))
	if err == nil || !strings.Contains(err.Error(), "DOCTYPE") {
		t.Fatalf("DTD must be rejected (XXE hardening), got %v", err)
	}
}

func TestOccurrenceBoundsPreserved(t *testing.T) {
	doc := `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="http://x" targetNamespace="http://x">
	  <wsdl:types><xs:schema targetNamespace="http://x">
	    <xs:complexType name="L"><xs:choice minOccurs="0" maxOccurs="unbounded">
	      <xs:element name="a" type="xs:string"/>
	    </xs:choice></xs:complexType>
	  </xs:schema></wsdl:types>
	</wsdl:definitions>`
	def, err := wsdl.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	ct := def.Schemas[0].Types["L"]
	if ct == nil || len(ct.Parts) != 1 {
		t.Fatalf("missing choice particle: %+v", def.Schemas[0].Types)
	}
	p := ct.Parts[0]
	if p.Kind != "choice" || p.Max != "unbounded" {
		t.Fatalf("bounds lost: %+v", p)
	}
	if !wsdl.Many(p.Min, p.Max) {
		t.Fatal("Many() must report unbounded")
	}
}
