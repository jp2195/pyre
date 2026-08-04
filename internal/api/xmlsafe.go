package api

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// decodeXML parses XML from r into v, rejecting DOCTYPE directives and
// any inline entity declarations. All API responses MUST be decoded
// through this function.
func decodeXML(r io.Reader, v any) error {
	dec := xml.NewDecoder(r)
	dec.Strict = true
	// Leave Entity nil — default map only contains the five predefined
	// XML entities (lt, gt, amp, quot, apos). Any <!ENTITY ...> in the
	// stream is raw token-visible and we reject it below.
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return fmt.Errorf("decodeXML: no element found")
		}
		if err != nil {
			return fmt.Errorf("decodeXML: %w", err)
		}
		switch t := tok.(type) {
		case xml.Directive:
			// Matches <!DOCTYPE ...>, <!ENTITY ...>, etc.
			return fmt.Errorf("decodeXML: rejecting xml directive (possible doctype/entity): %q", truncate(string(t), 64))
		case xml.ProcInst:
			// <?xml ...?> and friends — allowed, skip.
			continue
		case xml.CharData:
			// whitespace before root — skip.
			continue
		case xml.Comment:
			continue
		case xml.StartElement:
			return dec.DecodeElement(v, &t)
		}
	}
}

// DecodeXML is the exported form of decodeXML for sibling packages that
// parse PAN-OS XML outside the Client request path (keygen). It enforces
// the same hardening: XML directives (DOCTYPE) are rejected.
func DecodeXML(r io.Reader, v any) error {
	return decodeXML(r, v)
}

// InnerText decodes raw innerxml (Result.Inner) as plain text, unwrapping
// CDATA sections and resolving XML entities.
//
// PAN-OS returns free-form command output — `show system disk-space` (df),
// `show system resources` (top) — wrapped in CDATA. Because Result.Inner is
// captured with `xml:",innerxml"` it holds the *raw* bytes, so the literal
// "<![CDATA[" marker is still glued to the first line. Treating those bytes
// as text makes the first line unrecognizable to header checks. Always route
// free-form op output through this instead of string(resp.Result.Inner).
//
// Decoding goes through decodeXML, so the same DOCTYPE/entity hardening
// applies. On a decode failure the raw bytes are returned so a malformed
// response degrades to the previous behavior rather than losing all output.
func InnerText(inner []byte) string {
	var v struct {
		Data string `xml:",chardata"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(inner)), &v); err != nil {
		return string(inner)
	}
	return v.Data
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
