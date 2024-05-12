package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"os/exec"
	"pgStartTestTask/internal/storage/model"
	"pgStartTestTask/internal/storage/service"
	"strconv"
	"strings"
	"time"
)

const errKilled = "signal: killed"

type server struct {
	router  *mux.Router
	logger  *logrus.Logger
	tracker service.Tracker
	ctxs    map[int]context.CancelFunc
	cmds    map[int]Process
}
type Process interface {
	Kill() error
}

func newServer(tracker service.Tracker) *server {
	s := &server{
		router:  mux.NewRouter(),
		logger:  logrus.New(),
		tracker: tracker,
		ctxs:    make(map[int]context.CancelFunc),
		cmds:    make(map[int]Process),
	}

	s.configureRouter()

	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) configureRouter() {
	s.router.HandleFunc("/commands", s.CreateCommand()).Methods(http.MethodPost)
	s.router.HandleFunc("/commands", s.GetCommands()).Methods(http.MethodGet)
	s.router.HandleFunc("/commands/{id}", s.GetCommand()).Methods(http.MethodGet)
	s.router.HandleFunc("/commands/{id}", s.StopCommand()).Methods(http.MethodDelete)
}

func (s *server) StopCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid id"))
			s.logger.Info(mux.Vars(r))
			s.logger.Error("Error parsing command ID:", err)
			return
		}
		command, err := s.tracker.Command().GetById(id)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("undefined id"))
			s.logger.Error("Error getting command by ID:", err)
			return
		}
		switch command.Status {
		case "completed":
			s.error(w, r, http.StatusBadRequest, errors.New("script already completed"))
			s.logger.Info("script already completed")
			return
		case "failed":
			s.error(w, r, http.StatusBadRequest, errors.New("script already failed"))
			s.logger.Info("script already failed")
			return
		case "stopped":
			s.error(w, r, http.StatusBadRequest, errors.New("script already stopped"))
			s.logger.Info("script already stopped")
			return
		default:
			scripts := strings.Split(command.Script, ";")
			cansel, ok := s.ctxs[id]
			if !ok && len(scripts) > 1 {
				s.error(w, r, http.StatusInternalServerError, errors.New("Ooops..."))
				s.logger.Error("not in ctx")
				return
			}
			process, ok := s.cmds[id]
			if !ok {
				s.error(w, r, http.StatusInternalServerError, errors.New("Ooops..."))
				s.logger.Error("not in cmds")
				return
			}
			if len(scripts) > 1 {
				cansel()
			}
			if &process != nil {
				err := process.Kill()

				if err != nil {
					s.error(w, r, http.StatusInternalServerError, errors.New("Ooops..."))
					s.logger.Error("problem with kill")
					return
				}

				s.respond(w, r, http.StatusCreated, "Commands stopped")
			}

		}
	}
}

func (s *server) CreateCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var command *model.Command
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&command)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			s.logger.Error("Error decoding JSON:", err)
			return
		}
		if command.Script == "" {
			http.Error(w, "Empty Script", http.StatusBadRequest)
			s.logger.Error("Empty script")
			return
		}

		// Split the script into individual commands
		commands := strings.Split(command.Script, ";")
		newCommand := &model.Command{
			Script: strings.Join(commands, ";"),
			Output: "",
			Status: "in process",
		}
		if err := s.tracker.Command().Create(newCommand); err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			s.logger.Error("Error on SQL:", err)
			return
		}
		if len(commands) == 1 {
			ch := make(chan bool)
			go func(ch chan bool) {
				cmd := exec.Command("bash", "-c", commands[0])
				s.cmds[newCommand.Id] = cmd.Process
				out, err := cmd.CombinedOutput()
				if err != nil {
					if err.Error() == errKilled {
						newCommand.Status = "stopped"
						s.logger.Info("Command execution stopped")
						err = s.tracker.Command().Update(newCommand)
						if err != nil {
							s.error(w, r, http.StatusInternalServerError, err)
							s.logger.Error("Error on SQL:", err)
							ch <- true
							return
						}
					} else {
						newCommand.Status = "failed"
						s.logger.Errorf("Command execution failed: %v", err)
					}
				} else {
					newCommand.Status = "in process"
				}
				newCommand.Output = string(out)
				newCommand.Status = "completed"
				err = s.tracker.Command().Update(newCommand)
				if err != nil {
					s.error(w, r, http.StatusInternalServerError, err)
					s.logger.Error("Error on SQL:", err)
					return
				}
				ch <- true
			}(ch)
			select {
			case <-ch:
			case <-time.After(3 * time.Second):
			}
			err = s.tracker.Command().Update(newCommand)
			if err != nil {
				s.error(w, r, http.StatusInternalServerError, err)
				s.logger.Error("Error on SQL:", err)
				return
			}

			s.respond(w, r, http.StatusCreated, newCommand)
		} else {

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			s.ctxs[newCommand.Id] = cancel

			go func(commands []string) {
				for _, script := range commands {
					select {
					case <-ctx.Done():
						if newCommand.Status == "in process" {
							newCommand.Status = "stopped"
						}
						s.logger.Info("script is stopped in create")
						break
					default:
						script = strings.TrimSpace(script)
						if script == "" {
							continue
						}
						cmd := exec.Command("bash", "-c", script)
						s.cmds[newCommand.Id] = cmd.Process
						out, err := cmd.CombinedOutput()

						newCommand.Output += string(out)
						if err != nil {
							if err.Error() == errKilled {
								newCommand.Status = "stopped"
								s.logger.Info("Command execution stopped")
							} else {
								newCommand.Status = "failed"
								s.logger.Errorf("Command execution failed: %v", err)
							}
						} else {
							newCommand.Status = "in process"
						}

						// Add the command to the database
						err = s.tracker.Command().Update(newCommand)
						if err != nil {
							s.error(w, r, http.StatusInternalServerError, err)
							s.logger.Error("Error on SQL:", err)
							return
						}
					}
				}

				if newCommand.Status == "in process" {
					newCommand.Status = "completed"
				}
				err = s.tracker.Command().Update(newCommand)
				if err != nil {
					s.error(w, r, http.StatusInternalServerError, err)
					s.logger.Error("Error on SQL:", err)
					return
				}
			}(commands)

			s.respond(w, r, http.StatusCreated, newCommand)
		}
	}
}

func (s *server) GetCommands() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		commands, err := s.tracker.Command().GetList()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			s.logger.Error("Error getting commands:", err)
			return
		}
		s.respond(w, r, http.StatusOK, commands)
	}
}
func (s *server) GetCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid id"))
			s.logger.Error("Error parsing command ID:", err)
			return
		}
		command, err := s.tracker.Command().GetById(id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, errors.New("undefined id"))
			s.logger.Error("Error getting command by ID:", err)
			return
		}
		s.respond(w, r, http.StatusOK, command)
	}
}

func (s *server) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}

func (s *server) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
