package api

import (
	"bytes"
	"strings"
	"testing"
)

type sampleDoc struct {
	Status string `xml:"status,attr"`
	Value  string `xml:"value"`
}

func TestDecodeXML_OK(t *testing.T) {
	var out sampleDoc
	body := []byte(`<response status="success"><value>hi</value></response>`)
	if err := decodeXML(bytes.NewReader(body), &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != "success" || out.Value != "hi" {
		t.Fatalf("unexpected decode: %+v", out)
	}
}

func TestDecodeXML_RejectsDoctype(t *testing.T) {
	var out sampleDoc
	body := []byte(`<?xml version="1.0"?><!DOCTYPE foo><response status="x"><value>y</value></response>`)
	err := decodeXML(bytes.NewReader(body), &out)
	if err == nil || !strings.Contains(err.Error(), "doctype") {
		t.Fatalf("expected doctype rejection, got %v", err)
	}
}

func TestDecodeXML_RejectsInternalEntity(t *testing.T) {
	var out sampleDoc
	body := []byte(`<?xml version="1.0"?><!DOCTYPE lol [<!ENTITY a "AAAA">]><response status="x"><value>&a;</value></response>`)
	err := decodeXML(bytes.NewReader(body), &out)
	if err == nil {
		t.Fatalf("expected error on internal entity, got nil")
	}
}

func TestDecodeXML_Exported_RejectsDoctype(t *testing.T) {
	var out sampleDoc
	body := []byte(`<!DOCTYPE response [<!ENTITY x "boom">]><response status="success"><value>hi</value></response>`)
	err := DecodeXML(bytes.NewReader(body), &out)
	if err == nil || !strings.Contains(err.Error(), "doctype") {
		t.Fatalf("expected doctype rejection from exported DecodeXML, got %v", err)
	}
}
