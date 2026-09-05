package api

import (
	"bytes"
	"strings"
	"testing"
)

// PAN-OS reports parameter validation errors as <result><msg>text</msg>,
// but every other error as <msg><line>text</line>. Both must reach the user.
func TestErrorMessage_BothPANOSShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "parameter error under result",
			body: `<response status = 'error' code = '400'><result><msg>Illegal value for parameter nlogs [5001]. Should be between 1 to 5000</msg></result></response>`,
			want: "Illegal value for parameter nlogs [5001]. Should be between 1 to 5000",
		},
		{
			name: "job error under msg line",
			body: `<response status="error"><msg><line>No such query job</line></msg></response>`,
			want: "No such query job",
		},
		{
			name: "success carries no message",
			body: `<response status="success"><result><job>7</job></result></response>`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp XMLResponse
			if err := decodeXML(bytes.NewReader([]byte(tt.body)), &resp); err != nil {
				t.Fatalf("decodeXML: %v", err)
			}
			if got := resp.ErrorMessage(); got != tt.want {
				t.Errorf("ErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// CheckResponse must carry the parameter-error sentence, not just the code.
func TestCheckResponse_KeepsParameterErrorText(t *testing.T) {
	body := `<response status = 'error' code = '400'><result><msg>Illegal value for parameter nlogs [5001]. Should be between 1 to 5000</msg></result></response>`
	var resp XMLResponse
	if err := decodeXML(bytes.NewReader([]byte(body)), &resp); err != nil {
		t.Fatalf("decodeXML: %v", err)
	}
	err := CheckResponse(&resp)
	if err == nil {
		t.Fatal("CheckResponse returned nil for an error response")
	}
	if !strings.Contains(err.Error(), "Illegal value for parameter nlogs") {
		t.Errorf("error text lost the device message: %q", err.Error())
	}
}
