package httpapi

import (
	"bytes"
	"encoding/json"
	dead "github.com/example/hl7v2-message-router/internal/deadletter/application"
	delivery "github.com/example/hl7v2-message-router/internal/delivery/application"
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/application"
	routing "github.com/example/hl7v2-message-router/internal/routing/application"
	trace "github.com/example/hl7v2-message-router/internal/trace/application"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagementMessageFlowAndIdempotency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(hl7.NewParser(), routing.NewStore(), delivery.New(delivery.MemoryConnector{}), dead.New(), trace.New(), logger)
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	post := func(path string, value any) (int, map[string]any) {
		body, _ := json.Marshal(value)
		res, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out := map[string]any{}
		json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	if status, _ := post("/v1/targets", map[string]any{"ID": "ehr", "Name": "EHR", "Address": "memory://ehr"}); status != 201 {
		t.Fatalf("target status %d", status)
	}
	if status, _ := post("/v1/routes", map[string]any{"ID": "adt", "Name": "ADT", "MessageType": "ADT", "Trigger": "A01", "TargetIDs": []string{"ehr"}}); status != 201 {
		t.Fatalf("route status %d", status)
	}
	if status, _ := post("/v1/routes/adt/validate", map[string]any{}); status != 200 {
		t.Fatalf("validate status %d", status)
	}
	if status, _ := post("/v1/routes/adt/publish", map[string]any{}); status != 200 {
		t.Fatalf("publish status %d", status)
	}
	raw := "MSH|^~\\&|LAB|HOSP|EHR|HOSP|20260821150000||ADT^A01|MSGAPI1|P|2.5\rPID|1||12345||DOE^JANE"
	status, created := post("/v1/messages", map[string]string{"raw": raw})
	if status != 202 {
		t.Fatalf("message status %d: %#v", status, created)
	}
	id, _ := created["id"].(string)
	if id != "MSGAPI1" {
		t.Fatalf("id = %q", id)
	}
	status, duplicate := post("/v1/messages", map[string]string{"raw": raw})
	if status != 200 || duplicate["duplicate"] != true {
		t.Fatalf("duplicate = %d %#v", status, duplicate)
	}
	res, err := http.Get(ts.URL + "/v1/messages/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(data), "delivered") {
		t.Fatalf("query = %d %s", res.StatusCode, data)
	}
}
