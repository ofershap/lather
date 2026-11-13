package gen_test

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ofershap/lather/gen"
	"github.com/ofershap/lather/wsdl"
)

const choiceWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/" xmlns:tns="http://example.com/test" targetNamespace="http://example.com/test">
  <wsdl:types>
    <xs:schema targetNamespace="http://example.com/test" elementFormDefault="qualified">
      <xs:complexType name="mediaCollection"><xs:sequence><xs:element name="id" type="xs:string"/></xs:sequence></xs:complexType>
      <xs:complexType name="mediaMetadata"><xs:sequence><xs:element name="id" type="xs:string"/></xs:sequence></xs:complexType>
      <xs:complexType name="mediaList">
        <xs:choice minOccurs="0" maxOccurs="unbounded">
          <xs:element name="mediaCollection" type="tns:mediaCollection"/>
          <xs:element name="mediaMetadata" type="tns:mediaMetadata"/>
        </xs:choice>
      </xs:complexType>
      <xs:element name="GetList"><xs:complexType><xs:sequence><xs:element name="list" type="tns:mediaList"/></xs:sequence></xs:complexType></xs:element>
    </xs:schema>
  </wsdl:types>
  <wsdl:message name="GetListRequest"><wsdl:part name="parameters" element="tns:GetList"/></wsdl:message>
  <wsdl:portType name="TestPort"><wsdl:operation name="GetList"><wsdl:input message="tns:GetListRequest"/></wsdl:operation></wsdl:portType>
</wsdl:definitions>`

// Regression test for hooklift/gowsdl#264: xs:choice with
// maxOccurs="unbounded" must generate slice fields. gowsdl generates
// single pointer fields, silently dropping occurrences at unmarshal
// time. This is the failure lather exists to fix.
func TestChoiceUnboundedGeneratesSlices(t *testing.T) {
	def, err := wsdl.Parse(strings.NewReader(choiceWSDL))
	if err != nil {
		t.Fatal(err)
	}
	code, err := gen.Generate(def, gen.Options{Package: "test"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(code)
	norm := strings.Join(strings.Fields(src), " ")
	for _, want := range []string{"MediaCollection []MediaCollection", "MediaMetadata []MediaMetadata"} {
		if !strings.Contains(norm, want) {
			t.Fatalf("choice member %q must be a slice; generated:\n%s", want, src)
		}
	}
}

// writeTestModule writes a go.mod that resolves the lather module to
// the local source tree, so generated clients compile in isolation.
func writeTestModule(t *testing.T, dir string) {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	mod := "module genchk\n\ngo 1.21\n\nrequire github.com/ofershap/lather v0.0.0\n\nreplace github.com/ofershap/lather => " + root + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Generated code must compile.
func TestGeneratedCodeCompiles(t *testing.T) {
	def, err := wsdl.Parse(strings.NewReader(choiceWSDL))
	if err != nil {
		t.Fatal(err)
	}
	code, err := gen.Generate(def, gen.Options{Package: "genchk"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeTestModule(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "genchk.go"), code, 0o644); err != nil {
		t.Fatal(err)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code does not compile: %v\n%s\n\n%s", err, out, code)
	}
}

// Round-trip: unmarshalling a list with the generated types must keep
// every occurrence. gowsdl's output loses all but the last of each type.
func TestGeneratedRoundTripNoDataLoss(t *testing.T) {
	def, err := wsdl.Parse(strings.NewReader(choiceWSDL))
	if err != nil {
		t.Fatal(err)
	}
	code, err := gen.Generate(def, gen.Options{Package: "genchk"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeTestModule(t, dir)
	os.WriteFile(filepath.Join(dir, "genchk.go"), code, 0o644)
	testSrc := `package genchk

import (
	"encoding/xml"
	"testing"
)

func TestNoLoss(t *testing.T) {
	data := ` + "`" + `<list xmlns="http://example.com/test">
	  <mediaCollection><id>a</id></mediaCollection>
	  <mediaMetadata><id>b</id></mediaMetadata>
	  <mediaCollection><id>c</id></mediaCollection>
	</list>` + "`" + `
	var ml MediaList
	if err := xml.Unmarshal([]byte(data), &ml); err != nil {
		t.Fatal(err)
	}
	if len(ml.MediaCollection) != 2 || len(ml.MediaMetadata) != 1 {
		t.Fatalf("data loss: got %d collections, %d metadata", len(ml.MediaCollection), len(ml.MediaMetadata))
	}
	if ml.MediaCollection[0].Id != "a" || ml.MediaCollection[1].Id != "c" {
		t.Fatalf("order/values wrong: %+v", ml.MediaCollection)
	}
}
`
	os.WriteFile(filepath.Join(dir, "genchk_test.go"), []byte(testSrc), 0o644)
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("round-trip test failed: %v\n%s", err, out)
	}
}

var _ = xml.Name{}
