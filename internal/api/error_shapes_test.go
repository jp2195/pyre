package api

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
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

// captureDebugLog turns on PYRE_DEBUG-style tracing and captures what the
// standard logger receives for the duration of a test.
func captureDebugLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer

	prevFlag := debugLogging
	prevOut := log.Writer()
	prevFlags := log.Flags()
	debugLogging = true
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		debugLogging = prevFlag
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// The response trace has to read the same accessor the user-facing error
// does. It read Msg.Line only, so a parameter error -- the shape PAN-OS uses
// for exactly the arguments this client sends, such as an out-of-range nlogs
// -- traced as an empty string. Debug output that goes blank on the one class
// of error someone turns debugging on to investigate is worse than no line.
func TestRequest_DebugTraceKeepsParameterErrorText(t *testing.T) {
	buf := captureDebugLog(t)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<response status = 'error' code = '400'><result><msg>Illegal value for parameter nlogs [5001]. Should be between 1 to 5000</msg></result></response>`)
	})

	if _, err := c.Op(context.Background(), "<show><system><info></info></system></show>", ""); err != nil {
		t.Fatalf("Op: %v", err)
	}
	// Assert on the error line itself, not the whole buffer: the body
	// preview happens to repeat the message, which would hide a blank
	// error line.
	const prefix = "[API Response] error:"
	var errLine string
	for line := range strings.SplitSeq(buf.String(), "\n") {
		if strings.HasPrefix(line, prefix) {
			errLine = line
		}
	}
	if errLine == "" {
		t.Fatalf("no %q line in the trace:\n%s", prefix, buf.String())
	}
	if !strings.Contains(errLine, "Illegal value for parameter nlogs") {
		t.Errorf("the trace's error line dropped the device's message: %q", errLine)
	}
}
