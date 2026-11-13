// Package soap is a small SOAP 1.1/1.2 client runtime: envelope
// plumbing, WS-Security (UsernameToken + Timestamp), raw header
// injection, and fault decoding. Response parsing is hardened against
// DTD/XXE content.
package soap

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	NS11 = "http://schemas.xmlsoap.org/soap/envelope/"
	NS12 = "http://www.w3.org/2003/05/soap-envelope"
)

// Client calls a SOAP endpoint.
type Client struct {
	Endpoint string
	Version  string // "1.1" (default) or "1.2"
	HTTP     *http.Client
	// UsernameToken, when set, adds a WS-Security UsernameToken header
	// with a created timestamp (PasswordText).
	UsernameToken *UsernameToken
	// Headers are raw XML fragments injected into the SOAP Header
	// (WS-Addressing, vendor headers, signed sections produced
	// elsewhere). lather does not implement WS-Security signing.
	Headers []string
	// MaxResponseBytes caps the response body (default 32 MiB).
	MaxResponseBytes int64
}

// UsernameToken is a WS-Security UsernameToken with timestamp.
type UsernameToken struct {
	Username string
	Password string // sent as PasswordText (standard for TLS endpoints)
}

// Envelope is the SOAP envelope.
type Envelope struct {
	XMLName xml.Name
	Body    Body
}

// Body is the SOAP body.
type Body struct {
	XMLName xml.Name
	Fault   *Fault
	Payload []byte
}

// Fault is a decoded SOAP fault.
type Fault struct {
	Code   string
	String string
	Detail string
}

func (f *Fault) Error() string {
	return "soap: fault " + f.Code + ": " + f.String
}

type envelopeOut struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	EnvNS   string   `xml:"xmlns:soap,attr"`
	Header  *headerOut `xml:"soap:Header"`
	Body    bodyOut    `xml:"soap:Body"`
}

type headerOut struct {
	Inner string `xml:",innerxml"`
}

type bodyOut struct {
	Payload []byte `xml:",innerxml"`
}

type envelopeIn struct {
	Header struct {
		Inner []byte `xml:",innerxml"`
	}
	Body struct {
		Fault  *faultIn `xml:"Fault"`
		Inner  []byte   `xml:",innerxml"`
	}
}

type faultIn struct {
	Code   string `xml:"faultcode"`
	String string `xml:"faultstring"`
	Detail struct {
		Inner []byte `xml:",innerxml"`
	} `xml:"detail"`
	// SOAP 1.2 shapes
	Code12 struct {
		Value string `xml:"Value"`
	} `xml:"Code"`
	Reason struct {
		Text string `xml:"Text"`
	} `xml:"Reason"`
}

// Call marshals request (any struct) as the body payload, POSTs it, and
// unmarshals the response body into response (a pointer to a struct).
func (c *Client) Call(ctx context.Context, action string, request, response any) error {
	payload, err := xml.Marshal(request)
	if err != nil {
		return fmt.Errorf("soap: marshal request: %w", err)
	}

	ns := NS11
	if c.Version == "1.2" {
		ns = NS12
	}
	env := envelopeOut{EnvNS: ns, Body: bodyOut{Payload: payload}}
	var headers []string
	if c.UsernameToken != nil {
		headers = append(headers, c.UsernameToken.xml())
	}
	headers = append(headers, c.Headers...)
	if len(headers) > 0 {
		env.Header = &headerOut{Inner: strings.Join(headers, "")}
	}

	data, err := xml.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("soap: marshal envelope: %w", err)
	}

	contentType := "text/xml; charset=utf-8"
	if c.Version == "1.2" {
		contentType = `application/soap+xml; charset=utf-8; action="` + action + `"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	if c.Version != "1.2" && action != "" {
		req.Header.Set("SOAPAction", action)
	}

	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("soap: post: %w", err)
	}
	defer resp.Body.Close()

	max := c.MaxResponseBytes
	if max == 0 {
		max = 32 << 20
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, max))
	if err != nil {
		return fmt.Errorf("soap: read response: %w", err)
	}
	if bytes.Contains(body, []byte("<!DOCTYPE")) {
		return fmt.Errorf("soap: response contains DTD/DOCTYPE (refused, XXE hardening)")
	}

	var ein envelopeIn
	if err := xml.Unmarshal(body, &ein); err != nil {
		return fmt.Errorf("soap: decode response (%d): %w", resp.StatusCode, err)
	}
	if f := ein.Body.Fault; f != nil {
		code, str := f.Code, f.String
		if f.Code12.Value != "" {
			code = f.Code12.Value
		}
		if f.Reason.Text != "" {
			str = f.Reason.Text
		}
		return &Fault{Code: code, String: str, Detail: string(f.Detail.Inner)}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("soap: HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	if response != nil && len(ein.Body.Inner) > 0 {
		if err := xml.Unmarshal(ein.Body.Inner, response); err != nil {
			return fmt.Errorf("soap: decode payload: %w", err)
		}
	}
	return nil
}

func (t *UsernameToken) xml() string {
	var b strings.Builder
	b.WriteString(`<wsse:Security xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">`)
	b.WriteString(`<wsu:Timestamp wsu:Id="TS-lather"><wsu:Created>`)
	b.WriteString(time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	b.WriteString(`</wsu:Created></wsu:Timestamp>`)
	b.WriteString(`<wsse:UsernameToken><wsse:Username>`)
	xml.EscapeText(&b, []byte(t.Username))
	b.WriteString(`</wsse:Username><wsse:Password Type="PasswordText">`)
	xml.EscapeText(&b, []byte(t.Password))
	b.WriteString(`</wsse:Password></wsse:UsernameToken></wsse:Security>`)
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
