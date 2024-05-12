package apiserver

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"pgStartTestTask/internal/storage/model"
	"pgStartTestTask/internal/storage/service"
	"strings"
	"testing"
)

const total_return = "[{\"id\":1,\"script\":\"ls -l\",\"status\":\"complete\",\"output\":\"\",\"created_at\":null},{\"id\":2,\"script\":\"ls -l\",\"status\":\"failed\",\"output\":\"\",\"created_at\":null},{\"id\":3,\"script\":\"ls -l\",\"status\":\"in process\",\"output\":\"\",\"created_at\":null},{\"id\":4,\"script\":\"ls -l\",\"status\":\"stopped\",\"output\":\"\",\"created_at\":null}]\n"

type MockTracker struct{}

func (m *MockTracker) Command() service.CommandRepository {
	return &MockCommandRepository{}
}

type MockCommandRepository struct{}

func (m *MockCommandRepository) Create(command *model.Command) error {
	return nil
}
func (m *MockCommandRepository) Update(command *model.Command) error {
	return nil
}
func (m *MockCommandRepository) GetList() ([]model.Command, error) {
	return []model.Command{{Id: 1, Script: "ls -l", Status: "complete"},
			{Id: 2, Script: "ls -l", Status: "failed"},
			{Id: 3, Script: "ls -l", Status: "in process"},
			{Id: 4, Script: "ls -l", Status: "stopped"}},
		nil
}
func (m *MockCommandRepository) GetById(id int) (*model.Command, error) {
	switch id {
	case 1:
		return &model.Command{Id: 1, Script: "ls -l", Status: "completed"}, nil
	case 2:
		return &model.Command{Id: 2, Script: "ls -l", Status: "failed"}, nil
	case 3:
		return &model.Command{Id: 3, Script: "ls -l", Status: "in process"}, nil
	default:
		return &model.Command{Id: 4, Script: "ls -l", Status: "stopped"}, nil
	}
}

type APITestCase struct {
	Name         string
	Method, URL  string
	Body         string
	Header       http.Header
	WantStatus   int
	WantResponse string
}

func Endpoint(t *testing.T, server *server, tc APITestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		req, _ := http.NewRequest(tc.Method, tc.URL, bytes.NewBufferString(tc.Body))
		if tc.Header != nil {
			req.Header = tc.Header
		}
		res := httptest.NewRecorder()
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
		server.ServeHTTP(res, req)
		assert.Equal(t, tc.WantStatus, res.Code, "status mismatch")
		if tc.WantResponse != "" {
			pattern := strings.Trim(tc.WantResponse, "*")
			if pattern != tc.WantResponse {
				assert.Contains(t, res.Body.String(), pattern, "response mismatch")
			} else {
				assert.JSONEq(t, tc.WantResponse, res.Body.String(), "response mismatch")
			}
		}
	})
}
func TestAPI(t *testing.T) {
	tracker := &MockTracker{}
	serv := newServer(tracker)
	serv.ctxs[3] = func() {}
	cmd := exec.Command("bash", "-c", "sleep 100")
	serv.cmds[3] = cmd
	cmd.Start()

	tests := []APITestCase{
		{"invalid_id", http.MethodDelete, "/commands/invalid", "", nil, http.StatusBadRequest, "{\"error\":\"invalid id\"}\n"},
		{"stop complete", http.MethodDelete, "/commands/1", "", nil, http.StatusBadRequest, "{\"error\":\"script already completed\"}\n"},
		{"stop in process", http.MethodDelete, "/commands/3", "", nil, http.StatusCreated, `*Commands stopped*`},
		{"get all commands", http.MethodGet, "/commands", "", nil, http.StatusOK, total_return},
		{"get command by id", http.MethodGet, "/commands/2", "", nil, http.StatusOK, `*failed*`},
		{"get command by invalid id", http.MethodGet, "/commands/invalid", "", nil, http.StatusBadRequest, "{\"error\":\"invalid id\"}\n"},
		{"create command", http.MethodPost, "/commands", `{"script":"echo 'Hello, World!'","status":"in process"}`, nil, http.StatusCreated, "*Hello, World!*"},
		{"create command with empty script", http.MethodPost, "/commands", `{"script":"","status":"in process"}`, nil, http.StatusBadRequest, `*Empty Script*`},
	}
	for _, tc := range tests {
		Endpoint(t, serv, tc)
	}
}
