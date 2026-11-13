// Command lather generates Go client code from a WSDL 1.1 document.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ofershap/lather/gen"
	"github.com/ofershap/lather/wsdl"
)

func main() {
	var (
		pkg  = flag.String("p", "myservice", "package name for generated code")
		out  = flag.String("o", "", "output file (default <package>/<package>.go)")
		dir  = flag.String("d", ".", "directory under which the package directory is created")
		skip = flag.Bool("i", false, "skip TLS verification when fetching WSDL (insecure)")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "lather — generate Go clients from WSDL 1.1\n\nusage: lather [options] <service.wsdl|https://.../service?wsdl>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	src := flag.Arg(0)
	var r io.Reader
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		hc := &http.Client{Timeout: 30 * time.Second}
		if *skip {
			hc.Transport = insecureTransport()
		}
		resp, err := hc.Get(src)
		if err != nil {
			fatal("fetch WSDL: ", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			fatal("", fmt.Errorf("fetch WSDL: HTTP %d", resp.StatusCode))
		}
		r = io.LimitReader(resp.Body, 32<<20)
	} else {
		f, err := os.Open(src)
		if err != nil {
			fatal("open: ", err)
		}
		defer f.Close()
		r = f
	}

	def, err := wsdl.Parse(r)
	if err != nil {
		fatal("", err)
	}
	code, err := gen.Generate(def, gen.Options{Package: *pkg})
	if err != nil {
		fatal("generate: ", err)
	}

	outPath := *out
	if outPath == "" {
		outPath = filepath.Join(*dir, *pkg, *pkg+".go")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fatal("", err)
	}
	if err := os.WriteFile(outPath, code, 0o644); err != nil {
		fatal("", err)
	}
	fmt.Println("wrote", outPath)
}

func fatal(prefix string, err error) {
	fmt.Fprintln(os.Stderr, "lather:", prefix+errString(err))
	os.Exit(1)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
