package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

type MockPANOS struct {
	Server     *httptest.Server
	Hostname   string
	Model      string
	Serial     string
	Version    string
	IsPanorama bool
}

func NewMockPANOS() *MockPANOS {
	m := &MockPANOS{
		Hostname:   "mock-firewall",
		Model:      "PA-VM",
		Serial:     "007200001234",
		Version:    "10.2.3",
		IsPanorama: false,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", m.handleAPI)

	m.Server = httptest.NewTLSServer(mux)
	return m
}

func NewMockPanorama() *MockPANOS {
	m := &MockPANOS{
		Hostname:   "mock-panorama",
		Model:      "Panorama",
		Serial:     "007200009999",
		Version:    "10.2.3",
		IsPanorama: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", m.handleAPI)

	m.Server = httptest.NewTLSServer(mux)
	return m
}

func (m *MockPANOS) Close() {
	m.Server.Close()
}

func (m *MockPANOS) Host() string {
	return strings.TrimPrefix(m.Server.URL, "https://")
}

func (m *MockPANOS) handleAPI(w http.ResponseWriter, r *http.Request) {
	// Parse form for POST requests (keygen uses POST with form body)
	if r.Method == http.MethodPost {
		r.ParseForm()
	}

	// Get type from query string or form
	apiType := r.URL.Query().Get("type")
	if apiType == "" {
		apiType = r.FormValue("type")
	}
	cmd := r.URL.Query().Get("cmd")

	w.Header().Set("Content-Type", "application/xml")

	switch apiType {
	case "keygen":
		m.handleKeygen(w, r)
	case "op":
		m.handleOp(w, r, cmd)
	case "config":
		m.handleConfig(w, r)
	default:
		w.Write([]byte(`<response status="error"><msg><line>Invalid request</line></msg></response>`))
	}
}

func (m *MockPANOS) handleKeygen(w http.ResponseWriter, r *http.Request) {
	// Get user/password from query string or form (POST uses form body)
	user := r.URL.Query().Get("user")
	if user == "" {
		user = r.FormValue("user")
	}
	password := r.URL.Query().Get("password")
	if password == "" {
		password = r.FormValue("password")
	}

	if user == "admin" && password == "admin" {
		w.Write([]byte(`<response status="success"><result><key>LUFRPT1234567890abcdef==</key></result></response>`))
	} else {
		w.Write([]byte(`<response status="error"><msg><line>Invalid credentials</line></msg></response>`))
	}
}

func (m *MockPANOS) handleOp(w http.ResponseWriter, r *http.Request, cmd string) {
	switch {
	case strings.Contains(cmd, "<show><system><info>"):
		m.respondSystemInfo(w)
	case strings.Contains(cmd, "<show><system><resources>"):
		m.respondResources(w)
	case strings.Contains(cmd, "<show><session><info>"):
		m.respondSessionInfo(w)
	case strings.Contains(cmd, "<show><session><all>"):
		m.respondSessions(w)
	case strings.Contains(cmd, "<show><high-availability><state>"):
		m.respondHAStatus(w)
	case strings.Contains(cmd, "<show><interface>"):
		m.respondInterfaces(w)
	case strings.Contains(cmd, "<show><rule-hit-count>"):
		m.respondRuleHitCount(w)
	case strings.Contains(cmd, "<show><counter><global>"):
		m.respondThreatCounters(w)
	case strings.Contains(cmd, "<show><global-protect-gateway>"):
		m.respondGlobalProtect(w)
	case strings.Contains(cmd, "<request><license><info>"):
		m.respondLicenseInfo(w)
	case strings.Contains(cmd, "<show><devices><all>"):
		m.respondManagedDevices(w)
	default:
		w.Write([]byte(`<response status="success"><result></result></response>`))
	}
}

func (m *MockPANOS) handleConfig(w http.ResponseWriter, r *http.Request) {
	xpath := r.URL.Query().Get("xpath")

	if strings.Contains(xpath, "security/rules") {
		m.respondSecurityRules(w)
	} else {
		w.Write([]byte(`<response status="success"><result></result></response>`))
	}
}

func (m *MockPANOS) respondSystemInfo(w http.ResponseWriter) {
	fmt.Fprintf(w, `<response status="success">
<result>
<system>
<hostname>%s</hostname>
<model>%s</model>
<serial>%s</serial>
<sw-version>%s</sw-version>
<uptime>15 days, 3:42:18</uptime>
<devicename>%s</devicename>
<multi-vsys>off</multi-vsys>
<operational-mode>normal</operational-mode>
</system>
</result>
</response>`, m.Hostname, m.Model, m.Serial, m.Version, m.Hostname)
}

func (m *MockPANOS) respondResources(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
top - 14:32:18 up 15 days,  3:42,  0 users,  load average: 0.45, 0.52, 0.48
Tasks: 150 total,   1 running, 149 sleeping,   0 stopped,   0 zombie
%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.1 id,  0.3 wa,  0.0 hi,  0.3 si,  0.0 st
KiB Mem:  16384000 total, 12288000 used,  4096000 free,   256000 buffers
</result>
</response>`))
}

func (m *MockPANOS) respondSessionInfo(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<num-active>15432</num-active>
<num-max>262144</num-max>
<kbps>524288</kbps>
<cps>1250</cps>
</result>
</response>`))
}

func (m *MockPANOS) respondSessions(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<entry>
<idx>12345</idx>
<vsys>vsys1</vsys>
<application>web-browsing</application>
<state>ACTIVE</state>
<type>FLOW</type>
<source>192.168.1.100</source>
<sport>54321</sport>
<dst>8.8.8.8</dst>
<dport>443</dport>
<from>trust</from>
<to>untrust</to>
<xsource>10.0.0.100</xsource>
<xsport>54321</xsport>
<proto>6</proto>
<security-rule>allow-outbound</security-rule>
<start-time>Mon Jan 20 14:30:00 2025</start-time>
<total-byte-count>1048576</total-byte-count>
</entry>
<entry>
<idx>12346</idx>
<vsys>vsys1</vsys>
<application>ssl</application>
<state>ACTIVE</state>
<type>FLOW</type>
<source>192.168.1.101</source>
<sport>54322</sport>
<dst>1.1.1.1</dst>
<dport>443</dport>
<from>trust</from>
<to>untrust</to>
<xsource>10.0.0.101</xsource>
<xsport>54322</xsport>
<proto>6</proto>
<security-rule>allow-outbound</security-rule>
<start-time>Mon Jan 20 14:31:00 2025</start-time>
<total-byte-count>524288</total-byte-count>
</entry>
<entry>
<idx>12347</idx>
<vsys>vsys1</vsys>
<application>dns</application>
<state>ACTIVE</state>
<type>FLOW</type>
<source>192.168.1.102</source>
<sport>54323</sport>
<dst>8.8.4.4</dst>
<dport>53</dport>
<from>trust</from>
<to>untrust</to>
<xsource>10.0.0.102</xsource>
<xsport>54323</xsport>
<proto>17</proto>
<security-rule>allow-dns</security-rule>
<start-time>Mon Jan 20 14:32:00 2025</start-time>
<total-byte-count>2048</total-byte-count>
</entry>
</result>
</response>`))
}

func (m *MockPANOS) respondHAStatus(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<enabled>yes</enabled>
<group>
<local-info>
<state>active</state>
</local-info>
<peer-info>
<state>passive</state>
</peer-info>
<running-sync-enabled>yes</running-sync-enabled>
<running-sync>synchronized</running-sync>
</group>
</result>
</response>`))
}

func (m *MockPANOS) respondInterfaces(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<ifnet>
<entry>
<name>ethernet1/1</name>
<zone>untrust</zone>
<ip>203.0.113.1/24</ip>
<state>up</state>
<speed>1000</speed>
</entry>
<entry>
<name>ethernet1/2</name>
<zone>trust</zone>
<ip>192.168.1.1/24</ip>
<state>up</state>
<speed>1000</speed>
</entry>
<entry>
<name>ethernet1/3</name>
<zone>dmz</zone>
<ip>10.0.0.1/24</ip>
<state>up</state>
<speed>1000</speed>
</entry>
<entry>
<name>ethernet1/4</name>
<zone></zone>
<ip></ip>
<state>down</state>
<speed></speed>
</entry>
</ifnet>
<hw>
<entry>
<name>ethernet1/1</name>
<state>up</state>
<speed>1000</speed>
<duplex>full</duplex>
<mac>00:1b:17:00:01:01</mac>
</entry>
<entry>
<name>ethernet1/2</name>
<state>up</state>
<speed>1000</speed>
<duplex>full</duplex>
<mac>00:1b:17:00:01:02</mac>
</entry>
<entry>
<name>ethernet1/3</name>
<state>up</state>
<speed>1000</speed>
<duplex>full</duplex>
<mac>00:1b:17:00:01:03</mac>
</entry>
<entry>
<name>ethernet1/4</name>
<state>down</state>
<speed></speed>
<duplex></duplex>
<mac>00:1b:17:00:01:04</mac>
</entry>
</hw>
</result>
</response>`))
}

func (m *MockPANOS) respondSecurityRules(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<entry name="allow-outbound">
<disabled>no</disabled>
<action>allow</action>
<from><member>trust</member></from>
<to><member>untrust</member></to>
<source><member>any</member></source>
<destination><member>any</member></destination>
<application><member>web-browsing</member><member>ssl</member></application>
<service><member>application-default</member></service>
<log-end>yes</log-end>
</entry>
<entry name="allow-dns">
<disabled>no</disabled>
<action>allow</action>
<from><member>trust</member></from>
<to><member>untrust</member></to>
<source><member>any</member></source>
<destination><member>any</member></destination>
<application><member>dns</member></application>
<service><member>application-default</member></service>
<log-end>yes</log-end>
</entry>
<entry name="deny-all">
<disabled>no</disabled>
<action>deny</action>
<from><member>any</member></from>
<to><member>any</member></to>
<source><member>any</member></source>
<destination><member>any</member></destination>
<application><member>any</member></application>
<service><member>any</member></service>
<log-end>yes</log-end>
</entry>
<entry name="deprecated-rule">
<disabled>yes</disabled>
<action>allow</action>
<from><member>trust</member></from>
<to><member>dmz</member></to>
<source><member>any</member></source>
<destination><member>any</member></destination>
<application><member>any</member></application>
<service><member>any</member></service>
<log-end>no</log-end>
</entry>
</result>
</response>`))
}

func (m *MockPANOS) respondRuleHitCount(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<rule-hit-count>
<vsys>
<entry name="vsys1">
<rule-base>
<entry name="security">
<rules>
<entry name="allow-outbound">
<hit-count>1543289</hit-count>
<last-hit-timestamp>1737456000</last-hit-timestamp>
</entry>
<entry name="allow-dns">
<hit-count>892341</hit-count>
<last-hit-timestamp>1737455900</last-hit-timestamp>
</entry>
<entry name="deny-all">
<hit-count>12453</hit-count>
<last-hit-timestamp>1737455800</last-hit-timestamp>
</entry>
<entry name="deprecated-rule">
<hit-count>0</hit-count>
<last-hit-timestamp>0</last-hit-timestamp>
</entry>
</rules>
</entry>
</rule-base>
</entry>
</vsys>
</rule-hit-count>
</result>
</response>`))
}

func (m *MockPANOS) respondThreatCounters(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<global>
<counters>
<entry>
<name>flow_threat_block</name>
<value>1247</value>
<rate>2</rate>
<aspect>threat</aspect>
<desc>Threats blocked</desc>
<severity>high</severity>
</entry>
<entry>
<name>flow_threat_alert</name>
<value>3891</value>
<rate>5</rate>
<aspect>threat</aspect>
<desc>Threats alerted</desc>
<severity>medium</severity>
</entry>
<entry>
<name>flow_threat_critical</name>
<value>23</value>
<rate>0</rate>
<aspect>threat</aspect>
<desc>Critical threats</desc>
<severity>critical</severity>
</entry>
<entry>
<name>flow_threat_high</name>
<value>156</value>
<rate>1</rate>
<aspect>threat</aspect>
<desc>High severity threats</desc>
<severity>high</severity>
</entry>
</counters>
</global>
</result>
</response>`))
}

func (m *MockPANOS) respondGlobalProtect(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<entry>
<username>jsmith</username>
<domain>CORP</domain>
<computer>LAPTOP-001</computer>
<client>GlobalProtect Agent</client>
<virtual-ip>10.100.0.15</virtual-ip>
<login-time>Jan 21 08:30:00</login-time>
</entry>
<entry>
<username>mjones</username>
<domain>CORP</domain>
<computer>LAPTOP-002</computer>
<client>GlobalProtect Agent</client>
<virtual-ip>10.100.0.16</virtual-ip>
<login-time>Jan 21 09:15:00</login-time>
</entry>
<entry>
<username>agarcia</username>
<domain>CORP</domain>
<computer>DESKTOP-003</computer>
<client>GlobalProtect Agent</client>
<virtual-ip>10.100.0.17</virtual-ip>
<login-time>Jan 21 07:45:00</login-time>
</entry>
</result>
</response>`))
}

func (m *MockPANOS) respondLicenseInfo(w http.ResponseWriter) {
	w.Write([]byte(`<response status="success">
<result>
<licenses>
<entry>
<feature>PA-VM</feature>
<description>PA-VM</description>
<expires>January 01, 2027</expires>
<expired>no</expired>
</entry>
<entry>
<feature>Threat Prevention</feature>
<description>Threat Prevention</description>
<expires>January 01, 2027</expires>
<expired>no</expired>
</entry>
<entry>
<feature>GlobalProtect Gateway</feature>
<description>GlobalProtect Gateway</description>
<expires>January 01, 2027</expires>
<expired>no</expired>
</entry>
<entry>
<feature>WildFire License</feature>
<description>WildFire License</description>
<expires>January 01, 2027</expires>
<expired>no</expired>
</entry>
</licenses>
</result>
</response>`))
}

func (m *MockPANOS) respondManagedDevices(w http.ResponseWriter) {
	if !m.IsPanorama {
		w.Write([]byte(`<response status="error"><msg><line>Command not available on this device</line></msg></response>`))
		return
	}
	w.Write([]byte(`<response status="success">
<result>
<devices>
<entry name="007200001001">
<serial>007200001001</serial>
<hostname>fw-branch-01</hostname>
<ip-address>10.0.1.1</ip-address>
<model>PA-3260</model>
<sw-version>10.2.3</sw-version>
<ha><state>active</state></ha>
<connected>yes</connected>
<device-group>Branch-Offices</device-group>
</entry>
<entry name="007200001002">
<serial>007200001002</serial>
<hostname>fw-branch-02</hostname>
<ip-address>10.0.1.2</ip-address>
<model>PA-3260</model>
<sw-version>10.2.3</sw-version>
<ha><state>passive</state></ha>
<connected>yes</connected>
<device-group>Branch-Offices</device-group>
</entry>
<entry name="007200001003">
<serial>007200001003</serial>
<hostname>fw-dc-01</hostname>
<ip-address>10.0.2.1</ip-address>
<model>PA-5260</model>
<sw-version>10.2.3</sw-version>
<ha><state>active</state></ha>
<connected>yes</connected>
<device-group>Data-Center</device-group>
</entry>
<entry name="007200001004">
<serial>007200001004</serial>
<hostname>fw-dc-02</hostname>
<ip-address>10.0.2.2</ip-address>
<model>PA-5260</model>
<sw-version>10.2.3</sw-version>
<ha><state>passive</state></ha>
<connected>no</connected>
<device-group>Data-Center</device-group>
</entry>
</devices>
</result>
</response>`))
}
