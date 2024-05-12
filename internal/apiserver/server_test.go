package apiserver

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"pgStartTestTask/internal/storage/model"
	"pgStartTestTask/internal/storage/service"
	"strings"
	"testing"
)

const total_return = "[{\"id\":1,\"script\":\"ls -l\",\"status\":\"complete\",\"output\":\"\",\"created_at\":null},{\"id\":2,\"script\":\"ls -l\",\"status\":\"failed\",\"output\":\"\",\"created_at\":null},{\"id\":3,\"script\":\"ls -l\",\"status\":\"in process\",\"output\":\"\",\"created_at\":null},{\"id\":4,\"script\":\"ls -l\",\"status\":\"stopped\",\"output\":\"\",\"created_at\":null}]\n"

// MockTracker это мок службы, предоставляющий заглушки для методов, необходимых для тестирования сервера.
type MockTracker struct{}

type MockProcess struct{}

func (m *MockProcess) Kill() error {
	return nil
}

func (m *MockTracker) Command() service.CommandRepository {
	return &MockCommandRepository{}
}

// MockCommandRepository это мок службы команд, предоставляющий заглушки для методов.
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
	serv.cmds[3] = &MockProcess{}

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

//func TestStopCommandHandler(t *testing.T) {
//	// Создаем мок службы трекера и сервера
//	tracker := &MockTracker{}
//	server := apiserver.NewServer(tracker)
//
//	// Сценарий 1: Попытка остановить команду с недопустимым идентификатором
//	req1, err := http.NewRequest(http.MethodDelete, "/commands/invalid_id", nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//	rr1 := httptest.NewRecorder()
//	handler1 := http.HandlerFunc(server.StopCommand())
//	handler1.ServeHTTP(rr1, req1)
//	if status := rr1.Code; status != http.StatusBadRequest {
//		t.Errorf("handler returned wrong status code: got %v want %v",
//			status, http.StatusBadRequest)
//	}
//
//	// Сценарий 2: Попытка остановить завершенную успешно команду
//	req2, err := http.NewRequest(http.MethodDelete, "/commands/1", nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//	rr2 := httptest.NewRecorder()
//	handler2 := http.HandlerFunc(server.StopCommand())
//	handler2.ServeHTTP(rr2, req2)
//	if status := rr2.Code; status != http.StatusBadRequest {
//		t.Errorf("handler returned wrong status code: got %v want %v, %s",
//			status, http.StatusBadRequest, rr2.Body.String())
//		rr2.Body.String()
//	}
//
//	// Сценарий 3: Попытка остановить завершенную с ошибкой команду
//	req3, err := http.NewRequest("DELETE", "/commands/2", nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//	rr3 := httptest.NewRecorder()
//	handler3 := http.HandlerFunc(server.StopCommand())
//	handler3.ServeHTTP(rr3, req3)
//	if status := rr3.Code; status != http.StatusBadRequest {
//		t.Errorf("handler returned wrong status code: got %v want %v",
//			status, http.StatusInternalServerError)
//	}
//
//	// Сценарий 4: Успешная остановка команды, находящейся в процессе выполнения
//	req4, err := http.NewRequest("DELETE", "/commands/3", nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//	rr4 := httptest.NewRecorder()
//	handler4 := http.HandlerFunc(server.StopCommand())
//	handler4.ServeHTTP(rr4, req4)
//	if status := rr4.Code; status != http.StatusCreated {
//		t.Errorf("handler returned wrong status code: got %v want %v, %s",
//			status, http.StatusCreated, rr2.Body.String())
//	}
//}
//
//func TestCreateCommandHandler(t *testing.T) {
//	tracker := &MockTracker{}
//	server := apiserver.NewServer(tracker)
//	command := &model.Command{Script: "ls -l"}
//	body, _ := json.Marshal(command)
//	req, err := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	rr := httptest.NewRecorder()
//	handler := http.HandlerFunc(server.CreateCommand())
//	handler.ServeHTTP(rr, req)
//
//	if status := rr.Code; status != http.StatusCreated {
//		t.Errorf("handler returned wrong status code: got %v want %v",
//			status, http.StatusCreated)
//	}
//
//	var responseCommand model.Command
//	err = json.Unmarshal(rr.Body.Bytes(), &responseCommand)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if responseCommand.Script != command.Script {
//		t.Errorf("handler returned unexpected body: got %v want %v",
//			responseCommand.Script, command.Script)
//	}
//}
//
//func TestGetCommandsHandler(t *testing.T) {
//	tracker := &MockTracker{}
//	server := apiserver.NewServer(tracker)
//	req, err := http.NewRequest("GET", "/commands", nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	rr := httptest.NewRecorder()
//	handler := http.HandlerFunc(server.GetCommands())
//	handler.ServeHTTP(rr, req)
//
//	if status := rr.Code; status != http.StatusOK {
//		t.Errorf("handler returned wrong status code: got %v want %v",
//			status, http.StatusOK)
//	}
//
//	var responseCommands []*model.Command
//	err = json.Unmarshal(rr.Body.Bytes(), &responseCommands)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if len(responseCommands) != 1 {
//		t.Errorf("handler returned unexpected number of commands: got %v want %v",
//			len(responseCommands), 1)
//	}
//}
//
//func TestGetCommandHandler(t *testing.T) {
//	tracker := &MockTracker{}
//	server := apiserver.NewServer(tracker)
//	req, err := http.NewRequest("GET", "/commands/1", nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	rr := httptest.NewRecorder()
//	handler := http.HandlerFunc(server.GetCommand())
//	handler.ServeHTTP(rr, req)
//
//	if status := rr.Code; status != http.StatusOK {
//		t.Errorf("handler returned wrong status code: got %v want %v",
//			status, http.StatusOK)
//	}
//
//	var responseCommand model.Command
//	err = json.Unmarshal(rr.Body.Bytes(), &responseCommand)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if responseCommand.Id != 1 {
//		t.Errorf("handler returned unexpected command ID: got %v want %v",
//			responseCommand.Id, 1)
//	}
//}
