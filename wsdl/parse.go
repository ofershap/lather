package wsdl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Parse reads a WSDL 1.1 document. The XML decoder is hardened: DTD
// content is rejected outright so no external entity can be resolved
// (WSDL documents are fetched from the network and must be treated as
// attacker-controlled).
func Parse(r io.Reader) (*Definitions, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if idx := bytes.Index(raw, []byte("<!DOCTYPE")); idx >= 0 {
		return nil, fmt.Errorf("wsdl: DTD/DOCTYPE is not allowed (XXE hardening)")
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var root wsdlDefinitions
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("wsdl: decode: %w", err)
	}
	return build(&root)
}

// ---- raw XML shapes ----

type wsdlDefinitions struct {
	XMLName   xml.Name
	Name      string        `xml:"name,attr"`
	TargetNS  string        `xml:"targetNamespace,attr"`
	Imports   []wsdlImport  `xml:"import"`
	Types     []xsSchema    `xml:"types>schema"`
	Messages  []wsdlMessage `xml:"message"`
	PortTypes []wsdlPort    `xml:"portType"`
	Bindings  []wsdlBinding `xml:"binding"`
	Services  []wsdlService `xml:"service"`
}

type wsdlImport struct {
	Namespace string `xml:"namespace,attr"`
	Location  string `xml:"location,attr"`
}

type xsSchema struct {
	TargetNS string          `xml:"targetNamespace,attr"`
	ElemForm string          `xml:"elementFormDefault,attr"`
	Imports  []xsImport      `xml:"import"`
	Includes []xsImport      `xml:"include"`
	Elements []xsElement     `xml:"element"`
	Complex  []xsComplexType `xml:"complexType"`
	Simple   []xsSimpleType  `xml:"simpleType"`
}

type xsImport struct {
	Namespace string `xml:"namespace,attr"`
	Location  string `xml:"schemaLocation,attr"`
}

type xsElement struct {
	Name     string         `xml:"name,attr"`
	Ref      string         `xml:"ref,attr"`
	Type     string         `xml:"type,attr"`
	Min      string         `xml:"minOccurs,attr"`
	Max      string         `xml:"maxOccurs,attr"`
	Nillable string         `xml:"nillable,attr"`
	Complex  *xsComplexType `xml:"complexType"`
	Simple   *xsSimpleType  `xml:"simpleType"`
}

type xsComplexType struct {
	Name       string        `xml:"name,attr"`
	Doc        string        `xml:"annotation>documentation"`
	Sequence   *xsCompositor `xml:"sequence"`
	Choice     *xsCompositor `xml:"choice"`
	All        *xsCompositor `xml:"all"`
	Attributes []xsAttribute `xml:"attribute"`
	ExtContent *xsExtContent `xml:"complexContent>extension"`
	ResContent *xsExtContent `xml:"complexContent>restriction"`
	SimpleExt  *xsSimpleExt  `xml:"simpleContent>extension"`
	SimpleRes  *xsSimpleExt  `xml:"simpleContent>restriction"`
}

type xsExtContent struct {
	Base       string        `xml:"base,attr"`
	Sequence   *xsCompositor `xml:"sequence"`
	Choice     *xsCompositor `xml:"choice"`
	All        *xsCompositor `xml:"all"`
	Attributes []xsAttribute `xml:"attribute"`
}

type xsSimpleExt struct {
	Base       string        `xml:"base,attr"`
	Attributes []xsAttribute `xml:"attribute"`
}

type xsCompositor struct {
	Min      string         `xml:"minOccurs,attr"`
	Max      string         `xml:"maxOccurs,attr"`
	Elements []xsElement    `xml:"element"`
	Seq      []xsCompositor `xml:"sequence"`
	Choice   []xsCompositor `xml:"choice"`
	All      []xsCompositor `xml:"all"`
}

type xsAttribute struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
	Use  string `xml:"use,attr"`
}

type xsSimpleType struct {
	Name        string  `xml:"name,attr"`
	Restriction xsRestr `xml:"restriction"`
	List        xsList  `xml:"list"`
}

type xsRestr struct {
	Base  string   `xml:"base,attr"`
	Enums []xsEnum `xml:"enumeration"`
}

type xsEnum struct {
	Value string `xml:"value,attr"`
}

type xsList struct {
	ItemType string `xml:"itemType,attr"`
}

type wsdlMessage struct {
	Name  string     `xml:"name,attr"`
	Parts []wsdlPart `xml:"part"`
}

type wsdlPart struct {
	Name    string `xml:"name,attr"`
	Element string `xml:"element,attr"`
	Type    string `xml:"type,attr"`
}

type wsdlPort struct {
	Name string   `xml:"name,attr"`
	Ops  []wsdlOp `xml:"operation"`
}

type wsdlOp struct {
	Name  string    `xml:"name,attr"`
	In    wsdlOpMsg `xml:"input"`
	Out   wsdlOpMsg `xml:"output"`
	Fault wsdlOpMsg `xml:"fault"`
}

type wsdlOpMsg struct {
	Message string `xml:"message,attr"`
}

type wsdlBinding struct {
	Name string          `xml:"name,attr"`
	Type string          `xml:"type,attr"`
	SOAP []soapBinding   `xml:"binding"`
	Ops  []wsdlBindingOp `xml:"operation"`
}

type soapBinding struct {
	XMLName   xml.Name
	Style     string `xml:"style,attr"`
	Transport string `xml:"transport,attr"`
}

type wsdlBindingOp struct {
	Name string       `xml:"name,attr"`
	In   soapBodyWrap `xml:"input"`
	SOAP []soapOp     `xml:"operation"`
}

type soapBodyWrap struct {
	Body []soapBody `xml:"body"`
}

type soapBody struct {
	Use string `xml:"use,attr"`
}

type soapOp struct {
	XMLName    xml.Name
	SOAPAction string `xml:"soapAction,attr"`
	Style      string `xml:"style,attr"`
}

type wsdlService struct {
	Name  string         `xml:"name,attr"`
	Ports []wsdlServPort `xml:"port"`
}

type wsdlServPort struct {
	Name    string        `xml:"name,attr"`
	Binding string        `xml:"binding,attr"`
	Address []soapAddress `xml:"address"`
}

type soapAddress struct {
	XMLName  xml.Name
	Location string `xml:"location,attr"`
}

// ---- build the model ----

func build(root *wsdlDefinitions) (*Definitions, error) {
	d := &Definitions{
		Name:      root.Name,
		TargetNS:  root.TargetNS,
		Messages:  map[string]*Message{},
		PortTypes: map[string]*PortType{},
		Bindings:  map[string]*Binding{},
		Services:  map[string]*Service{},
	}
	for _, im := range root.Imports {
		d.Imports = append(d.Imports, im.Location)
	}
	for i := range root.Types {
		d.Schemas = append(d.Schemas, buildSchema(&root.Types[i]))
	}
	for _, m := range root.Messages {
		msg := &Message{Name: m.Name}
		for _, p := range m.Parts {
			msg.Parts = append(msg.Parts, MessagePart(p))
		}
		d.Messages[msg.Name] = msg
	}
	for _, pt := range root.PortTypes {
		p := &PortType{Name: pt.Name}
		for _, op := range pt.Ops {
			p.Operations = append(p.Operations, &Operation{
				Name: op.Name, Input: op.In.Message, Output: op.Out.Message, Fault: op.Fault.Message,
			})
		}
		d.PortTypes[p.Name] = p
	}
	for _, b := range root.Bindings {
		bi := &Binding{Name: b.Name, Type: b.Type, Style: "document"}
		for _, sb := range b.SOAP {
			if sb.Style != "" {
				bi.Style = sb.Style
			}
			bi.SOAPVer = soapVersion(sb.XMLName.Space)
		}
		for _, op := range b.Ops {
			bo := &BindingOp{Name: op.Name}
			for _, so := range op.SOAP {
				bo.SOAPAction = so.SOAPAction
				if so.XMLName.Space != "" {
					bi.SOAPVer = soapVersion(so.XMLName.Space)
				}
			}
			bi.Operations = append(bi.Operations, bo)
		}
		d.Bindings[bi.Name] = bi
	}
	for _, s := range root.Services {
		sv := &Service{Name: s.Name}
		for _, p := range s.Ports {
			port := &Port{Name: p.Name, Binding: p.Binding}
			for _, a := range p.Address {
				port.Address = a.Location
			}
			sv.Ports = append(sv.Ports, port)
		}
		d.Services[sv.Name] = sv
	}
	return d, nil
}

func soapVersion(ns string) string {
	switch {
	case strings.Contains(ns, "soap12"):
		return "1.2"
	case strings.Contains(ns, "soap/http") || strings.Contains(ns, "wsdl/soap/"):
		return "1.1"
	}
	return ""
}

func buildSchema(s *xsSchema) *Schema {
	sc := &Schema{
		TargetNS: s.TargetNS,
		ElemForm: s.ElemForm,
		Elements: map[string]*Element{},
		Types:    map[string]*ComplexType{},
		Simple:   map[string]*SimpleType{},
	}
	for _, im := range s.Imports {
		sc.Imports = append(sc.Imports, SchemaImport{Namespace: im.Namespace, Location: im.Location})
	}
	for _, im := range s.Includes {
		sc.Imports = append(sc.Imports, SchemaImport{Namespace: im.Namespace, Location: im.Location, Include: true})
	}
	for i := range s.Elements {
		el := buildElement(&s.Elements[i])
		sc.Elements[el.Name] = el
	}
	for i := range s.Complex {
		ct := buildComplex(&s.Complex[i])
		sc.Types[ct.Name] = ct
	}
	for i := range s.Simple {
		st := buildSimple(&s.Simple[i])
		sc.Simple[st.Name] = st
	}
	return sc
}

func buildElement(e *xsElement) *Element {
	el := &Element{
		Name: e.Name, Type: e.Type, Ref: e.Ref,
		Min: e.Min, Max: e.Max, Nillable: e.Nillable == "true" || e.Nillable == "1",
	}
	if e.Complex != nil {
		el.Inline = buildComplex(e.Complex)
	}
	if e.Simple != nil {
		el.InlineSm = buildSimple(e.Simple)
	}
	return el
}

func buildComplex(c *xsComplexType) *ComplexType {
	ct := &ComplexType{Name: c.Name, Doc: c.Doc}
	switch {
	case c.Sequence != nil:
		ct.Parts = append(ct.Parts, buildCompositor("sequence", c.Sequence))
	case c.Choice != nil:
		ct.Parts = append(ct.Parts, buildCompositor("choice", c.Choice))
	case c.All != nil:
		ct.Parts = append(ct.Parts, buildCompositor("all", c.All))
	}
	for _, a := range c.Attributes {
		ct.Attributes = append(ct.Attributes, &Attribute{Name: a.Name, Type: a.Type, Use: a.Use})
	}
	if c.ExtContent != nil {
		ct.Base, ct.BaseKind = c.ExtContent.Base, "extension"
		if c.ExtContent.Sequence != nil {
			ct.Parts = append(ct.Parts, buildCompositor("sequence", c.ExtContent.Sequence))
		}
		if c.ExtContent.Choice != nil {
			ct.Parts = append(ct.Parts, buildCompositor("choice", c.ExtContent.Choice))
		}
		if c.ExtContent.All != nil {
			ct.Parts = append(ct.Parts, buildCompositor("all", c.ExtContent.All))
		}
		for _, a := range c.ExtContent.Attributes {
			ct.Attributes = append(ct.Attributes, &Attribute{Name: a.Name, Type: a.Type, Use: a.Use})
		}
	}
	if c.ResContent != nil {
		ct.Base, ct.BaseKind = c.ResContent.Base, "restriction"
		if c.ResContent.Sequence != nil {
			ct.Parts = append(ct.Parts, buildCompositor("sequence", c.ResContent.Sequence))
		}
		for _, a := range c.ResContent.Attributes {
			ct.Attributes = append(ct.Attributes, &Attribute{Name: a.Name, Type: a.Type, Use: a.Use})
		}
	}
	if c.SimpleExt != nil {
		ct.SimpleContentBase = c.SimpleExt.Base
		for _, a := range c.SimpleExt.Attributes {
			ct.Attributes = append(ct.Attributes, &Attribute{Name: a.Name, Type: a.Type, Use: a.Use})
		}
	}
	if c.SimpleRes != nil {
		ct.SimpleContentBase = c.SimpleRes.Base
		for _, a := range c.SimpleRes.Attributes {
			ct.Attributes = append(ct.Attributes, &Attribute{Name: a.Name, Type: a.Type, Use: a.Use})
		}
	}
	return ct
}

// buildCompositor preserves occurrence bounds on compositors and elements.
// This is the exact place where naive generators collapse unbounded
// choices into single fields and silently drop data at unmarshal time;
// lather keeps Min/Max on every particle so the generator can always
// emit slices where the schema allows many.
func buildCompositor(kind string, c *xsCompositor) *Particle {
	p := &Particle{Kind: kind, Min: c.Min, Max: c.Max}
	for i := range c.Elements {
		el := buildElement(&c.Elements[i])
		p.Elements = append(p.Elements, el)
	}
	for i := range c.Seq {
		p.Children = append(p.Children, buildCompositor("sequence", &c.Seq[i]))
	}
	for i := range c.Choice {
		p.Children = append(p.Children, buildCompositor("choice", &c.Choice[i]))
	}
	for i := range c.All {
		p.Children = append(p.Children, buildCompositor("all", &c.All[i]))
	}
	return p
}

func buildSimple(s *xsSimpleType) *SimpleType {
	st := &SimpleType{Name: s.Name, Base: s.Restriction.Base, ListOf: s.List.ItemType}
	for _, e := range s.Restriction.Enums {
		st.Enums = append(st.Enums, e.Value)
	}
	return st
}
