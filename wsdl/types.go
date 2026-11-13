// Package wsdl parses WSDL 1.1 documents and their embedded or imported
// XML Schemas into a type model designed for correct Go code generation.
//
// The model is occurrence-aware: every particle keeps its min/max
// occurrence bounds so generators never collapse unbounded content into
// single-value fields (the class of bug where list data is silently
// dropped at unmarshal time).
package wsdl

// Definitions is the root of a parsed WSDL document.
type Definitions struct {
	Name      string
	TargetNS  string
	Schemas   []*Schema
	Messages  map[string]*Message
	PortTypes map[string]*PortType
	Bindings  map[string]*Binding
	Services  map[string]*Service
	Imports   []string // unresolved import locations (informational)
}

// Schema is one xs:schema block.
type Schema struct {
	TargetNS string
	ElemForm string // elementFormDefault: "qualified" or "" (unqualified)
	Elements map[string]*Element
	Types    map[string]*ComplexType
	Simple   map[string]*SimpleType
	Imports  []SchemaImport
}

// SchemaImport records an xs:import or xs:include.
type SchemaImport struct {
	Namespace string
	Location  string
	Include   bool
}

// Element is an xs:element (global or particle-local).
type Element struct {
	Name     string
	Type     string // qualified type name, may be empty if inline
	Inline   *ComplexType
	InlineSm *SimpleType
	Min      string // occurrence bounds, "" means "1"
	Max      string // "unbounded" or a number, "" means "1"
	Nillable bool
	Ref      string // qualified name when this is a ref to a global element
}

// ComplexType is an xs:complexType with ordered content particles.
type ComplexType struct {
	Name       string
	Doc        string
	Base       string // qualified base type for extension/restriction
	BaseKind   string // "extension", "restriction", or ""
	Parts      []*Particle
	Attributes []*Attribute
	// SimpleContent holds the base when this is a simpleContent extension
	// (text-bearing type with attributes).
	SimpleContentBase string
}

// Particle is one content node: an element, or a compositor
// (sequence/choice/all) holding elements (possibly nested compositors).
type Particle struct {
	Kind     string // "element", "sequence", "choice", "all"
	Elements []*Element
	Children []*Particle // nested compositors
	Min      string
	Max      string
}

// Attribute is an xs:attribute.
type Attribute struct {
	Name string
	Type string
	Use  string // "optional", "required", "prohibited"
}

// SimpleType is an xs:simpleType (restriction with optional enumerations).
type SimpleType struct {
	Name   string
	Base   string
	Enums  []string
	MinOcc string
	ListOf string // item type when defined via xs:list
}

// Message is a wsdl:message.
type Message struct {
	Name  string
	Parts []MessagePart
}

// MessagePart is one wsdl:part.
type MessagePart struct {
	Name    string
	Element string // qualified
	Type    string // qualified
}

// PortType is a wsdl:portType.
type PortType struct {
	Name       string
	Operations []*Operation
}

// Operation is a wsdl:operation.
type Operation struct {
	Name   string
	Input  string // qualified message name
	Output string
	Fault  string
}

// Binding is a wsdl:binding.
type Binding struct {
	Name       string
	Type       string // qualified portType name
	SOAPVer    string // "1.1" or "1.2"
	Style      string // "document" or "rpc"
	Operations []*BindingOp
}

// BindingOp is a binding-level operation with its SOAP action.
type BindingOp struct {
	Name       string
	SOAPAction string
}

// Service is a wsdl:service.
type Service struct {
	Name  string
	Ports []*Port
}

// Port is a wsdl:port with its endpoint address.
type Port struct {
	Name    string
	Binding string // qualified
	Address string
}

// MaxUnbounded reports whether an occurrence bound is "unbounded".
func MaxUnbounded(max string) bool { return max == "unbounded" }

// Many reports whether the bounds allow more than one occurrence.
func Many(min, max string) bool {
	if MaxUnbounded(max) {
		return true
	}
	switch max {
	case "", "0", "1":
		return false
	}
	return true
}
