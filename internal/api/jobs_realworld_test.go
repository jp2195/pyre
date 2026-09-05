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

// jobsResponse is trimmed from a real `show jobs all` on a PA-440 running
// 11.2.10-h8. Two details in it are not what the parser assumed:
//
//   - tdeq carries a time with no date ("21:43:45"), while tenq and tfin
//     carry full datetimes. Parsing tdeq on its own matches no layout, so
//     every job's start time came back as the zero time.
//   - progress holds a percentage only while a job is running. Once it
//     finishes, PAN-OS puts the completion timestamp in that field, so
//     parsing it as a number yielded 0% for every completed job.
const jobsResponse = `<response status="success"><result>
<job><tenq>2026/09/04 21:43:45</tenq><tdeq>21:43:45</tdeq><id>2122</id><user>admin</user>
<type>Commit</type><status>FIN</status><queued>NO</queued><stoppable>no</stoppable>
<result>OK</result><tfin>2026/09/04 21:45:56</tfin><progress>100</progress>
<details><line>Configuration committed successfully</line></details></job>
<job><tenq>2026/08/30 15:00:43</tenq><tdeq>15:00:44</tdeq><id>1995</id><user></user>
<type>WildFire</type><status>FIN</status><queued>NO</queued><stoppable>no</stoppable>
<result>OK</result><tfin>2026/08/30 15:01:04</tfin><progress>2026/08/30 15:01:04</progress>
<details></details></job>
<job><tenq>2026/08/30 23:59:50</tenq><tdeq>00:00:05</tdeq><id>1990</id><user></user>
<type>Downld</type><status>FIN</status><queued>NO</queued><stoppable>no</stoppable>
<result>OK</result><tfin>2026/08/31 00:00:30</tfin><progress>2026/08/31 00:00:30</progress>
<details></details></job>
</result></response>`

func jobsClient(t *testing.T) *api.Client {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, jobsResponse)
	}))
	t.Cleanup(srv.Close)
	c, err := api.NewClient(strings.TrimPrefix(srv.URL, "https://"), "k", api.ClientOptions{Insecure: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetJobs_ParsesDatelessStartTime(t *testing.T) {
	jobs, err := jobsClient(t).GetJobs(context.Background(), "")
	if err != nil {
		t.Fatalf("GetJobs: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("got %d jobs, want 3", len(jobs))
	}

	byID := map[int]int{}
	for i, j := range jobs {
		byID[j.ID] = i
	}

	commit := jobs[byID[2122]]
	if commit.StartTime.IsZero() {
		t.Error("commit job has no start time; tdeq carries a time with no date")
	}
	if got := commit.StartTime.Format("2006-01-02 15:04:05"); got != "2026-09-04 21:43:45" {
		t.Errorf("start time = %q, want 2026-09-04 21:43:45 (tdeq dated from tenq)", got)
	}
	if got := commit.EndTime.Format("2006-01-02 15:04:05"); got != "2026-09-04 21:45:56" {
		t.Errorf("end time = %q, want 2026-09-04 21:45:56", got)
	}
}

// TestGetJobs_StartTimeRollsOverMidnight covers the case that makes dating
// tdeq from tenq non-trivial: a job queued at 23:59 and started at 00:00
// belongs to the following day, not the previous one.
func TestGetJobs_StartTimeRollsOverMidnight(t *testing.T) {
	jobs, err := jobsClient(t).GetJobs(context.Background(), "")
	if err != nil {
		t.Fatalf("GetJobs: %v", err)
	}
	for _, j := range jobs {
		if j.ID != 1990 {
			continue
		}
		if got := j.StartTime.Format("2006-01-02 15:04:05"); got != "2026-08-31 00:00:05" {
			t.Errorf("start time = %q, want 2026-08-31 00:00:05 (queued 08-30 23:59:50)", got)
		}
		if j.StartTime.Before(j.EndTime) != true {
			t.Errorf("start %v is not before end %v", j.StartTime, j.EndTime)
		}
		return
	}
	t.Fatal("job 1990 missing")
}

// TestGetJobs_ProgressOfFinishedJob covers the field PAN-OS reuses: a
// completed job reports a timestamp where a percentage would go.
func TestGetJobs_ProgressOfFinishedJob(t *testing.T) {
	jobs, err := jobsClient(t).GetJobs(context.Background(), "")
	if err != nil {
		t.Fatalf("GetJobs: %v", err)
	}
	for _, j := range jobs {
		if j.Status != "FIN" {
			continue
		}
		if j.Progress != 100 {
			t.Errorf("job %d finished but reports %d%% progress", j.ID, j.Progress)
		}
	}
}
