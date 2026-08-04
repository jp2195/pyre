package api

import (
	"bytes"
	"context"
	"log"

	"github.com/jp2195/pyre/internal/models"
)

// addressEntry mirrors the PAN-OS XML <entry> shape under /address.
type addressEntry struct {
	Name        string `xml:"name,attr"`
	IPNetmask   string `xml:"ip-netmask"`
	IPRange     string `xml:"ip-range"`
	FQDN        string `xml:"fqdn"`
	IPWildcard  string `xml:"ip-wildcard"`
	Description string `xml:"description"`
	Tag         struct {
		Member []string `xml:"member"`
	} `xml:"tag"`
}

func parseAddressEntries(inner []byte) []addressEntry {
	// Try with <address> wrapper (xpath ends at /address).
	var withWrapper struct {
		Entry []addressEntry `xml:"address>entry"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(inner)), &withWrapper); err == nil && len(withWrapper.Entry) > 0 {
		return withWrapper.Entry
	}
	// Try without wrapper (entries directly in <result>).
	var withoutWrapper struct {
		Entry []addressEntry `xml:"entry"`
	}
	if decodeXML(bytes.NewReader(WrapInner(inner)), &withoutWrapper) == nil {
		return withoutWrapper.Entry
	}
	return nil
}

// fetchObjectsFromPath fetches XML config at one xpath, trying Show then Get.
// Returns parsed entries or an empty slice (never nil) on no-results / soft errors.
// Hard transport errors are returned to the caller.
func fetchObjectsFromPath[T any](
	c *Client, ctx context.Context, xpath, target string, parse func([]byte) []T,
) ([]T, error) {
	// Try Show first.
	resp, err := c.Show(ctx, xpath, target)
	if err == nil && resp.IsSuccess() && len(resp.Result.Inner) > 0 {
		entries := parse(resp.Result.Inner)
		if len(entries) > 0 {
			return entries, nil
		}
	}
	// Fall back to Get.
	resp, getErr := c.Get(ctx, xpath, target)
	if getErr != nil {
		// Both Show and Get failed; return the most recent error (Get) so the
		// caller sees the actual fallback failure, not the stale Show error.
		return nil, getErr
	}
	if !resp.IsSuccess() {
		return []T{}, nil
	}
	if len(resp.Result.Inner) == 0 {
		return []T{}, nil
	}
	entries := parse(resp.Result.Inner)
	if entries == nil {
		return []T{}, nil
	}
	return entries, nil
}

func convertAddressEntry(e addressEntry) (models.AddressObject, bool) {
	var typ, value string
	switch {
	case e.IPNetmask != "":
		typ, value = "ip-netmask", e.IPNetmask
	case e.IPRange != "":
		typ, value = "ip-range", e.IPRange
	case e.FQDN != "":
		typ, value = "fqdn", e.FQDN
	case e.IPWildcard != "":
		typ, value = "ip-wildcard", e.IPWildcard
	default:
		log.Printf("api: address object %q has no recognized type element; skipping", e.Name)
		return models.AddressObject{}, false
	}
	return models.AddressObject{
		Name:        e.Name,
		Type:        typ,
		Value:       value,
		Description: e.Description,
		Tags:        append([]string(nil), e.Tag.Member...),
	}, true
}

// GetAddresses fetches address objects from vsys1 and shared, concatenated.
func (c *Client) GetAddresses(ctx context.Context, target string) ([]models.AddressObject, error) {
	const (
		vsysXPath   = "/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/address"
		sharedXPath = "/config/shared/address"
	)

	vsysEntries, err := fetchObjectsFromPath(c, ctx, vsysXPath, target, parseAddressEntries)
	if err != nil {
		return nil, err
	}
	sharedEntries, err := fetchObjectsFromPath(c, ctx, sharedXPath, target, parseAddressEntries)
	if err != nil {
		return nil, err
	}

	out := make([]models.AddressObject, 0, len(vsysEntries)+len(sharedEntries))
	for _, e := range vsysEntries {
		if o, ok := convertAddressEntry(e); ok {
			out = append(out, o)
		}
	}
	for _, e := range sharedEntries {
		if o, ok := convertAddressEntry(e); ok {
			out = append(out, o)
		}
	}
	return out, nil
}

// serviceEntry mirrors the PAN-OS XML <entry> shape under /service.
type serviceEntry struct {
	Name     string `xml:"name,attr"`
	Protocol struct {
		TCP struct {
			Port       string `xml:"port"`
			SourcePort string `xml:"source-port"`
		} `xml:"tcp"`
		UDP struct {
			Port       string `xml:"port"`
			SourcePort string `xml:"source-port"`
		} `xml:"udp"`
	} `xml:"protocol"`
	Description string `xml:"description"`
	Tag         struct {
		Member []string `xml:"member"`
	} `xml:"tag"`
}

func parseServiceEntries(inner []byte) []serviceEntry {
	var withWrapper struct {
		Entry []serviceEntry `xml:"service>entry"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(inner)), &withWrapper); err == nil && len(withWrapper.Entry) > 0 {
		return withWrapper.Entry
	}
	var withoutWrapper struct {
		Entry []serviceEntry `xml:"entry"`
	}
	if decodeXML(bytes.NewReader(WrapInner(inner)), &withoutWrapper) == nil {
		return withoutWrapper.Entry
	}
	return nil
}

func convertServiceEntry(e serviceEntry) (models.ServiceObject, bool) {
	var proto, dest, src string
	switch {
	case e.Protocol.TCP.Port != "":
		proto, dest, src = "tcp", e.Protocol.TCP.Port, e.Protocol.TCP.SourcePort
	case e.Protocol.UDP.Port != "":
		proto, dest, src = "udp", e.Protocol.UDP.Port, e.Protocol.UDP.SourcePort
	default:
		log.Printf("api: service object %q has no recognized protocol element; skipping", e.Name)
		return models.ServiceObject{}, false
	}
	return models.ServiceObject{
		Name:        e.Name,
		Protocol:    proto,
		DestPort:    dest,
		SrcPort:     src,
		Description: e.Description,
		Tags:        append([]string(nil), e.Tag.Member...),
	}, true
}

// GetServices fetches service objects from vsys1 and shared, concatenated.
func (c *Client) GetServices(ctx context.Context, target string) ([]models.ServiceObject, error) {
	const (
		vsysXPath   = "/config/devices/entry[@name='localhost.localdomain']/vsys/entry[@name='vsys1']/service"
		sharedXPath = "/config/shared/service"
	)

	vsysEntries, err := fetchObjectsFromPath(c, ctx, vsysXPath, target, parseServiceEntries)
	if err != nil {
		return nil, err
	}
	sharedEntries, err := fetchObjectsFromPath(c, ctx, sharedXPath, target, parseServiceEntries)
	if err != nil {
		return nil, err
	}

	out := make([]models.ServiceObject, 0, len(vsysEntries)+len(sharedEntries))
	for _, e := range vsysEntries {
		if o, ok := convertServiceEntry(e); ok {
			out = append(out, o)
		}
	}
	for _, e := range sharedEntries {
		if o, ok := convertServiceEntry(e); ok {
			out = append(out, o)
		}
	}
	return out, nil
}
