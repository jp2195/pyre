package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/api"
)

// Some PAN-OS op commands answer with plain text rather than XML, and wrap it
// in CDATA. Nothing stripped that wrapper, so the first line of the payload
// still had "<![CDATA[" glued to its front. For disk usage that defeated the
// header check: the line does not start with "Filesystem", so the header was
// parsed as a filesystem, producing a phantom row called "Mounted" with 0%
// usage that pushed a real filesystem off the panel.
//
// Copied verbatim from a PA-440 running 11.2.10-h8.
const diskSpaceResponse = `<response status="success"><result><![CDATA[Filesystem      Size  Used Avail Use% Mounted on
/dev/mmcblk0p3   21G  6.0G   14G  31% /
none            7.7G   64K  7.7G   1% /dev
/dev/mmcblk0p5   32G   13G   18G  42% /opt/pancfg
/dev/mmcblk0p6   18G   14G  2.9G  83% /opt/panrepo
tmpfs           7.7G  3.8G  4.0G  49% /dev/shm
cgroup_root     7.7G     0  7.7G   0% /cgroup
/dev/mmcblk0p8   22G   11G   11G  50% /opt/panlogs
tmpfs            12M   48K   12M   1% /opt/pancfg/mgmt/ssl/private
]]></result></response>`

func plainTextClient(t *testing.T, body string) *api.Client {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	c, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetDiskUsage_SkipsTheHeaderInsideCDATA(t *testing.T) {
	disks, err := plainTextClient(t, diskSpaceResponse).GetDiskUsage(context.Background(), "")
	if err != nil {
		t.Fatalf("GetDiskUsage: %v", err)
	}

	for _, d := range disks {
		if strings.Contains(d.Filesystem, "CDATA") || d.MountPoint == "Mounted" {
			t.Errorf("header row parsed as a filesystem: %+v", d)
		}
	}
	if len(disks) != 8 {
		t.Errorf("got %d filesystems, want the 8 the device listed", len(disks))
	}

	byMount := map[string]float64{}
	for _, d := range disks {
		byMount[d.MountPoint] = d.Percent
	}
	for mount, want := range map[string]float64{
		"/":            31,
		"/opt/pancfg":  42,
		"/opt/panrepo": 83,
		"/opt/panlogs": 50,
	} {
		if got, ok := byMount[mount]; !ok {
			t.Errorf("filesystem %s missing", mount)
		} else if got != want {
			t.Errorf("%s = %.0f%%, want %.0f%%", mount, got, want)
		}
	}
}

// TestGetSystemResources_ReadsThroughCDATA covers the other plain-text
// response. It happened to work because it is parsed with regexes, but it
// should not depend on that.
func TestGetSystemResources_ReadsThroughCDATA(t *testing.T) {
	const body = `<response status="success"><result><![CDATA[top - 22:35:27 up 31 days, 11:06,  0 users,  load average: 4.38, 4.20, 4.11
Tasks: 150 total,   1 running, 149 sleeping,   0 stopped,   0 zombie
%Cpu(s):  3.1 us,  1.2 sy,  0.0 ni, 95.4 id,  0.3 wa,  0.0 hi,  0.0 si,  0.0 st
KiB Mem : 16384000 total,  7536000 used,  8848000 free,   256000 buffers
]]></result></response>`

	res, err := plainTextClient(t, body).GetSystemResources(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSystemResources: %v", err)
	}
	if res.Load1 != 4.38 {
		t.Errorf("Load1 = %v, want 4.38", res.Load1)
	}
	if res.CPUPercent < 4.2 || res.CPUPercent > 4.4 {
		t.Errorf("CPUPercent = %v, want us+sy = 4.3", res.CPUPercent)
	}
	if res.MemoryPercent < 45 || res.MemoryPercent > 47 {
		t.Errorf("MemoryPercent = %v, want ~46", res.MemoryPercent)
	}
}

// TestGetJobs_DeduplicatesRepeatedIDs covers device behavior rather than a
// parsing mistake: `show jobs all` returns some jobs twice, byte for byte.
// The Recent Jobs panel listed the same job twice, which reads as a bug.
func TestGetJobs_DeduplicatesRepeatedIDs(t *testing.T) {
	const body = `<response status="success"><result>
<job><tenq>2026/09/04 22:06:30</tenq><tdeq>22:06:30</tdeq><id>2127</id><type>Downld</type><status>FIN</status><result>OK</result><tfin>2026/09/04 22:06:37</tfin><progress>100</progress></job>
<job><tenq>2026/09/04 22:06:30</tenq><tdeq>22:06:30</tdeq><id>2127</id><type>Downld</type><status>FIN</status><result>OK</result><tfin>2026/09/04 22:06:37</tfin><progress>100</progress></job>
<job><tenq>2026/09/04 22:05:00</tenq><tdeq>22:05:00</tdeq><id>2126</id><type>WildFire</type><status>FIN</status><result>OK</result><tfin>2026/09/04 22:05:10</tfin><progress>100</progress></job>
</result></response>`

	jobs, err := plainTextClient(t, body).GetJobs(context.Background(), "")
	if err != nil {
		t.Fatalf("GetJobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2 distinct ones", len(jobs))
	}
	seen := map[int]bool{}
	for _, j := range jobs {
		if seen[j.ID] {
			t.Errorf("job %d listed more than once", j.ID)
		}
		seen[j.ID] = true
	}
}
