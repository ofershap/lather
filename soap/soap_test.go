package soap_test

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ofershap/lather/soap"
)

type addReq struct {
	XMLName xml.Name `xml:"Add"`
	A       int      `xml:"intA"`
	B       int      `xml:"intB"`
}

type addResp struct {
	XMLName    xml.Name `xml:"AddResponse"`
	AddResult  int      `xml:"AddResult"`
}

func server(t *testing.T, handler func(body []byte) string) (*httptest.Server, *[]byte) {
	t.Helper()
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = b
		w.Header().Set("Content-Type", "text/xml")
		io.WriteString(w, handler(b))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func TestCallRoundTrip(t *testing.T) {
	srv, got := server(t, func(body []byte) string {
		return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><AddResponse><AddResult>42</AddResult></AddResponse></soap:Body></soap:Envelope>`
	})
	c := &soap.Client{Endpoint: srv.URL}
	var out addResp
	if err := c.Call(context.Background(), "http://tempuri.org/Add", &addReq{A: 40, B: 2}, &out); err != nil {
		t.Fatal(err)
	}
	if out.AddResult != 42 {
		t.Fatalf("got %d", out.AddResult)
	}
	if !strings.Contains(string(*got), "<SOAPAction>" ) && !strings.Contains(string(*got), "<intA>40</intA>") {
		t.Fatalf("request payload missing: %s", *got)
	}
}

func TestFaultDecoded(t *testing.T) {
	srv, _ := server(t, func(body []byte) string {
		return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><soap:Fault><faultcode>soap:Server</faultcode><faultstring>division by zero</faultstring></soap:Fault></soap:Body></soap:Envelope>`
	})
	c := &soap.Client{Endpoint: srv.URL}
	err := c.Call(context.Background(), "Divide", &addReq{}, &addResp{})
	f, ok := err.(*soap.Fault)
	if !ok {
		t.Fatalf("want *soap.Fault, got %T: %v", err, err)
	}
	if !strings.Contains(f.String, "division by zero") {
		t.Fatalf("fault string: %q", f.String)
	}
}

func TestXXERefused(t *testing.T) {
	srv, _ := server(t, func(body []byte) string {
		return `<!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body/></soap:Envelope>`
	})
	c := &soap.Client{Endpoint: srv.URL}
	err := c.Call(context.Background(), "X", &addReq{}, &addResp{})
	if err == nil || !strings.Contains(err.Error(), "DTD") {
		t.Fatalf("XXE response must be refused, got %v", err)
	}
}

func TestUsernameTokenHeader(t *testing.T) {
	srv, got := server(t, func(body []byte) string {
		return `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><AddResponse><AddResult>1</AddResult></AddResponse></soap:Body></soap:Envelope>`
	})
	c := &soap.Client{Endpoint: srv.URL, UsernameToken: &soap.UsernameToken{Username: "u", Password: "p"}}
	var out addResp
	if err := c.Call(context.Background(), "Add", &addReq{}, &out); err != nil {
		t.Fatal(err)
	}
	s := string(*got)
	for _, want := range []string{"wsse:Security", "wsu:Timestamp", "<wsse:Username>u</wsse:Username>", "PasswordText"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in request:\n%s", want, s)
		}
	}
}

func TestSOAP12ContentType(t *testing.T) {
	var ct, action string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		action = r.Header.Get("SOAPAction")
		io.WriteString(w, `<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body><AddResponse><AddResult>1</AddResult></AddResponse></soap:Body></soap:Envelope>`)
	}))
	defer srv.Close()
	c := &soap.Client{Endpoint: srv.URL, Version: "1.2"}
	var out addResp
	if err := c.Call(context.Background(), "urn:Add", &addReq{}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ct, "application/soap+xml") || !strings.Contains(ct, `action="urn:Add"`) {
		t.Fatalf("SOAP 1.2 content type wrong: %q", ct)
	}
	if action != "" {
		t.Fatalf("SOAP 1.2 must not use SOAPAction header, got %q", action)
	}
}
