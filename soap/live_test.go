package soap_test

import (
	"context"
	"encoding/xml"
	"os"
	"testing"
	"time"

	"github.com/ofershap/lather/soap"
)

// TestLivePublicEndpoint is an opt-in integration test against a real
// public SOAP service. It is skipped unless LATHER_LIVE=1 is set, so CI
// and normal runs never depend on the network or a third-party service.
//
// Run it with: LATHER_LIVE=1 go test ./soap/ -run TestLive
func TestLivePublicEndpoint(t *testing.T) {
	if os.Getenv("LATHER_LIVE") != "1" {
		t.Skip("set LATHER_LIVE=1 to run the live integration test")
	}
	c := &soap.Client{
		Endpoint: "http://webservices.oorsprong.org/websamples.countryinfo/CountryInfoService.wso",
		Version:  "1.1",
	}
	type capitalCity struct {
		XMLName         xml.Name `xml:"http://www.oorsprong.org/websamples.countryinfo CapitalCity"`
		SCountryISOCode string   `xml:"sCountryISOCode"`
	}
	var out struct {
		CapitalCityResult string `xml:"CapitalCityResult"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := c.Call(ctx, "", &capitalCity{SCountryISOCode: "IL"}, &out); err != nil {
		t.Fatalf("live call: %v", err)
	}
	if out.CapitalCityResult != "Jerusalem" {
		t.Fatalf("unexpected capital: %q", out.CapitalCityResult)
	}
}
