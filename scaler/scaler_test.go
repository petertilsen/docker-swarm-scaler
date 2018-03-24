package scaler

import (
	"bytes"
	"net/http"
	"os"
	"testing"
)

var cu, ta = 1, 1

type mockResponse struct {
	t       *testing.T
	headers http.Header
	body    []byte
	status  int
}

func NewResponse(t *testing.T) *mockResponse {
	return &mockResponse{
		t:       t,
		headers: make(http.Header),
	}
}

func (r *mockResponse) Header() http.Header {
	return r.headers
}

func (r *mockResponse) Write(body []byte) (int, error) {
	r.body = body
	return len(body), nil
}

func (r *mockResponse) WriteHeader(status int) {
	r.status = status
}

func (r *mockResponse) Assert(status int, body string) {
	if r.status != status {
		r.t.Errorf("expected status %+v to equal %+v", r.status, status)
	}
	if string(r.body) != body {
		r.t.Errorf("expected body %+v to equal %+v", string(r.body), body)
	}
}

func TestHandlerBadRequest(t *testing.T) {
	reqNil := &http.Request{}
	res := NewResponse(t)
	handler(res, reqNil)
	if res.status != http.StatusBadRequest {
		t.Errorf("Expected %q, got %q", http.StatusBadRequest, res.status)
	}
}

func TestHandlerGoodRequest(t *testing.T) {
	res := NewResponse(t)
	s := &service{}
	origReplicas := getReplicas
	origCommand := executeCmd
	replicas = mockGetReplicas
	command = s.mockCommandDebug
	defer func() { command = origCommand }()
	defer func() { replicas = origReplicas }()

	json := []byte(`{"version": "4",
    "groupKey": "1",
    "status": "resolved",
    "receiver": "test",
    "groupLabels": "",
    "commonLabels": "",
    "commonAnnotations": "",
    "externalURL": "test",
    "alerts": [
    {
        "labels": {"summary": "test", "description": "test"},
        "annotations": {"service": "docker-tools_cadvisor", "summary": "test", "description": "test"},
        "startsAt": "2018-03-01T22:08:41+00:00"
	}
	]
	}`)
	req, _ := http.NewRequest("POST", "localhost", bytes.NewBuffer(json))

	handler(res, req)
	if res.status == http.StatusBadRequest {
		t.Errorf("Expected %q, got %q", http.StatusBadRequest, res.status)
	}
	e := `{"Service":[{"Name":"docker-tools_cadvisor","Scale":0}],"Status":"OK"}`
	if string(res.body) != e {
		t.Errorf("Expected %q, got %q", e, string(res.body))
	}
}

func TestHandleAlerts(t *testing.T) {
	an := map[string]string{
		"service": "test"}
	as := []alert{
		{
			Annotations: an}}

	a := &alerts{
		Status: "firing",
		Alerts: as}

	s := &service{}
	origReplicas := getReplicas
	origCommand := executeCmd
	replicas = mockGetReplicas
	command = s.mockCommandDebug
	defer func() { command = origCommand }()
	defer func() { replicas = origReplicas }()

	ss := a.handleAlerts()

	if len(ss.Service) == 0 {
		t.Errorf("Expected >0, got %q", len(ss.Service))
	}
}

func TestScaleUp(t *testing.T) {
	s := &service{
		Name:  "test",
		Scale: 1}

	origReplicas := getReplicas
	origCommand := executeCmd
	replicas = mockGetReplicas
	command = s.mockCommandDebug
	defer func() { command = origCommand }()
	defer func() { replicas = origReplicas }()

	r := s.scaleUp()
	if r != true {
		t.Errorf("Expected %t, got %t", true, r)
	}
	if s.Scale != 2 {
		t.Errorf("Expected %q, got %q", 2, s.Scale)
	}

	cu, ta = 50, 50
	rM := s.scaleUp()
	if rM != true {
		t.Errorf("Expected %t, got %t", true, rM)
	}

	cu, ta = 1, 2
	rF := s.scaleUp()
	if rF != true {
		t.Errorf("Expected %t, got %t", true, rF)
	}
	if s.Scale != 2 {
		t.Errorf("Expected %q, got %q", 2, s.Scale)
	}

}

func TestScaleDown(t *testing.T) {
	s := &service{
		Name:  "test",
		Scale: 1}

	origReplicas := getReplicas
	origCommand := executeCmd
	replicas = mockGetReplicas
	command = s.mockCommandDebug
	defer func() { command = origCommand }()
	defer func() { replicas = origReplicas }()

	rM := s.scaleDown()
	if rM != true {
		t.Errorf("Expected %t, got %t", true, rM)
	}
	if s.Scale != 1 {
		t.Errorf("Expected %q, got %q", 1, s.Scale)
	}

	cu, ta = 3, 3
	r := s.scaleDown()
	if r != true {
		t.Errorf("Expected %t, got %t", true, r)
	}
	if s.Scale != 2 {
		t.Errorf("Expected %q, got %q", 2, s.Scale)
	}

	cu, ta = 1, 2
	rC := s.scaleDown()
	if rC != true {
		t.Errorf("Expected %t, got %t", true, rC)
	}
	if s.Scale != 2 {
		t.Errorf("Expected %q, got %q", 2, s.Scale)
	}

}

func TestGetReplicas(t *testing.T) {
	s := &service{
		Name:  "test",
		Scale: 1}

	origCommand := executeCmd
	command = s.mockCommandDebug
	defer func() { command = origCommand }()

	e := "docker service ls --filter name=test --format '{{.Replicas}}'"

	rD := s.replicasCmd()
	if string(rD[:]) != e {
		t.Errorf("Expected %q, got %q", e, string(rD[:]))
	}

	command = s.mockCommand
	cu, ta := getReplicas(s)
	if cu != 1 || ta != 1 {
		t.Errorf("Expected %q %q, got %q %q", 1, 1, cu, ta)
	}

}

func TestLogin(t *testing.T) {
	s := &service{}

	os.Setenv("AWS_ACCESS_KEY_ID", "1")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1")
	origCommand := executeCmd
	command = s.mockCommandDebug
	defer func() { command = origCommand }()

	e := "export AWS_ACCESS_KEY_ID=1 && export AWS_SECRET_ACCESS_KEY=1 " +
		"&& login=`aws ecr get-login --no-include-email --region eu-central-1` && echo `$login`"

	r := s.loginCmd()
	if string(r[:]) != e {
		t.Errorf("Expected %q, got %q", e, string(r[:]))
	}
}

func TestImage(t *testing.T) {
	s := &service{
		Name: "test"}

	os.Setenv("AWS_ACCESS_KEY_ID", "1")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "1")
	origCommand := executeCmd
	command = s.mockCommandDebug
	defer func() { command = origCommand }()

	e := "docker service ls --filter name=test --format '{{.Image}}'"

	r := s.imageCmd()
	if string(r[:]) != e {
		t.Errorf("Expected %q, got %q", e, string(r[:]))
	}
}

func TestScale(t *testing.T) {
	s := &service{
		Name:  "test",
		Scale: 1}

	origCommand := executeCmd
	command = s.mockCommandDebug
	defer func() { command = origCommand }()

	e := "docker service ls --filter name=test --format '{{.Image}}'"

	r := s.imageCmd()
	if string(r[:]) != e {
		t.Errorf("Expected %q, got %q", e, string(r[:]))
	}
}

func TestUpdate(t *testing.T) {
	s := &service{
		Name: "test"}

	origCommand := executeCmd
	command = s.mockCommandDebug
	defer func() { command = origCommand }()

	e := "docker pull image " +
		"&& docker service update --force --update-parallelism 1 --update-delay 10s " +
		"--with-registry-auth --image image test"

	r := s.updateCmd("image")
	if string(r[:]) != e {
		t.Errorf("Expected %q, got %q", e, string(r[:]))
	}
}

func TestExecuteCommandDryRun(t *testing.T) {
	os.Setenv("DRYRUN", "true")
	r := executeCmd("", true)
	if len(r) != 0 {
		t.Errorf("Expected %q, got %q", 0, len(r))
	}
}

func TestExecuteCommand(t *testing.T) {
	r := executeCmd("echo \"test\"", false)
	if len(r) == 0 {
		t.Errorf("Expected >0, got %q", len(r))
	}
}

func mockGetReplicas(s *service) (int, int) {
	return cu, ta
}

func (s *service) mockCommandDebug(cmd string, dryrun bool) []byte {
	return []byte(cmd)
}

func (s *service) mockCommand(cmd string, dryrun bool) []byte {
	return []byte("1/1")
}
