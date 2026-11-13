# Security Policy

## Reporting a vulnerability

Please report security issues through GitHub's private vulnerability reporting
("Report a vulnerability" on the Security tab of this repository). Do not open
a public issue for a vulnerability.

## Scope notes

lather parses WSDL documents and SOAP responses, and generates Go source. It
refuses DTD/DOCTYPE in both directions (XXE hardening), caps response size
(32 MiB by default), and is standard-library only. The `-i` CLI flag disables
TLS verification when fetching a WSDL - use it only against services you
control.
