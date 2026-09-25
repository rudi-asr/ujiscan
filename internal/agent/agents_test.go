package agent

import (
	"testing"
)

// --- Recon XML parsing ---

func TestParseNmapXML(t *testing.T) {
	xmlData := `<?xml version="1.0"?>
<nmaprun scanner="nmap" version="7.99">
  <host>
    <status state="up" reason="syn-ack"/>
    <address addr="127.0.0.1" addrtype="ipv4"/>
    <hostnames><hostname name="localhost" type="PTR"/></hostnames>
    <os><osmatch name="Linux 2.6.32" accuracy="98"/></os>
    <ports>
      <port protocol="tcp" portid="22">
        <state state="open" reason="syn-ack"/>
        <service name="ssh" product="OpenSSH" version="9.0"/>
      </port>
      <port protocol="tcp" portid="80">
        <state state="closed" reason="reset"/>
        <service name="http"/>
      </port>
    </ports>
  </host>
</nmaprun>`

	hosts := parseNmapXML(xmlData)
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	h := hosts[0]
	if h.IP != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %q", h.IP)
	}
	if len(h.Ports) != 1 { // only open ports retained
		t.Fatalf("expected 1 open port, got %d", len(h.Ports))
	}
	if h.Ports[0].Service != "ssh" || h.Ports[0].Version != "9.0" {
		t.Errorf("unexpected service info: %+v", h.Ports[0])
	}
}

func TestParseNmapXMLMalformed(t *testing.T) {
	hosts := parseNmapXML("<not-xml>")
	if hosts == nil || len(hosts) != 0 {
		t.Fatalf("expected empty result for malformed XML, got %v", hosts)
	}
}

// --- Nuclei JSONL parsing ---

func TestParseNucleiJSONL(t *testing.T) {
	data := `{"template-id":"xss-test","type":"http","matched-at":"http://127.0.0.1/","host":"127.0.0.1","info":{"name":"Reflected XSS","severity":"high"}}
garbage-line
{"template-id":"sql-test","type":"http","matched-at":"http://127.0.0.1/search","info":{"name":"SQL Injection","severity":"critical"}}`

	vulns := parseNucleiJSONL(data)
	if len(vulns) != 2 {
		t.Fatalf("expected 2 valid vulns (garbage skipped), got %d", len(vulns))
	}
	if vulns[0].Name != "Reflected XSS" || vulns[0].Severity != "high" {
		t.Errorf("unexpected first vuln: %+v", vulns[0])
	}
	if vulns[1].TemplateID != "sql-test" {
		t.Errorf("unexpected second vuln: %+v", vulns[1])
	}
}

// --- Analyzer classification ---

func TestClassifyFinding(t *testing.T) {
	f := classifyFinding(map[string]interface{}{
		"title":       "SQL Injection in login",
		"severity":    "Critical",
		"description": "Injectable parameter",
	})
	if f == nil {
		t.Fatal("classifyFinding returned nil")
	}
	if f.Severity != "critical" {
		t.Errorf("expected severity critical, got %q", f.Severity)
	}
	if f.CVSS != "9.8" {
		t.Errorf("expected CVSS 9.8 for critical, got %q", f.CVSS)
	}
	if f.Remediation == "" {
		t.Error("expected non-empty remediation")
	}
}

func TestClassifyFindingEmpty(t *testing.T) {
	if f := classifyFinding(map[string]interface{}{}); f != nil {
		t.Errorf("expected nil for empty finding, got %+v", f)
	}
}

func TestNormalizeSeverity(t *testing.T) {
	cases := map[string]string{
		"Critical": "critical",
		"HIGH":     "high",
		"Moderate": "medium",
		"1":        "low",
		"0":        "info",
		"weird":    "",
	}
	for in, want := range cases {
		if got := normalizeSeverity(in); got != want {
			t.Errorf("normalizeSeverity(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- Validation ---

func TestReconValidate(t *testing.T) {
	r := NewReconnaissanceAgent(nil)
	if err := r.Validate(&Task{EngagementID: "eng-1", Params: map[string]interface{}{"target": "127.0.0.1"}}); err != nil {
		t.Errorf("valid recon task rejected: %v", err)
	}
	if err := r.Validate(&Task{EngagementID: "eng-1"}); err == nil {
		t.Error("recon task without target should fail validation")
	}
	if err := r.Validate(&Task{Params: map[string]interface{}{"target": "127.0.0.1"}}); err == nil {
		t.Error("recon task without engagement_id should fail validation")
	}
}

func TestScannerValidate(t *testing.T) {
	s := NewScannerAgent(nil)
	if err := s.Validate(&Task{EngagementID: "eng-1", Params: map[string]interface{}{"target_url": "http://127.0.0.1"}}); err != nil {
		t.Errorf("valid scanner task rejected: %v", err)
	}
	if err := s.Validate(&Task{EngagementID: "eng-1"}); err == nil {
		t.Error("scanner task without target should fail validation")
	}
}

// --- Param helpers ---

func TestParamHelpers(t *testing.T) {
	params := map[string]interface{}{
		"target": "127.0.0.1",
		"port":   float64(8081),
		"fast":   true,
	}
	if got := getStringParam(params, "target"); got != "127.0.0.1" {
		t.Errorf("getStringParam = %q", got)
	}
	if got := getIntParam(params, "port", 0); got != 8081 {
		t.Errorf("getIntParam (float64) = %d", got)
	}
	if got := getIntParam(params, "missing", 42); got != 42 {
		t.Errorf("getIntParam default = %d", got)
	}
	if !getBoolParam(params, "fast", false) {
		t.Error("getBoolParam should be true")
	}
}

// --- Queue: duplicate IDs & cancel ---

func TestQueueRejectsDuplicateID(t *testing.T) {
	q := NewTaskQueue(3, 10)
	if err := q.Enqueue(&Task{ID: "t1"}); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}
	if err := q.Enqueue(&Task{ID: "t1"}); err == nil {
		t.Fatal("duplicate task ID should be rejected")
	}
}

func TestQueueCancelSkipsTerminated(t *testing.T) {
	q := NewTaskQueue(3, 10)
	_ = q.Enqueue(&Task{ID: "cancel-me", Priority: 100})
	_ = q.Enqueue(&Task{ID: "run-me", Priority: 50})

	task := q.GetTask("cancel-me")
	if task == nil {
		t.Fatal("cancel-me task not found")
	}
	task.Status = StatusTerminated // what Manager.CancelTask does

	first := q.Dequeue()
	if first == nil || first.ID != "run-me" {
		t.Fatalf("expected run-me to be dequeued first (cancelled task skipped), got %+v", first)
	}
	if q.GetTask("cancel-me") != nil {
		t.Error("cancelled task should be purged from active tasks")
	}
}
