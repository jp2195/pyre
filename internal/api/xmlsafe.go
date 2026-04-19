package api

import (
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
