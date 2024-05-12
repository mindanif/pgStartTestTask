package service

import "pgStartTestTask/internal/storage/model"

type CommandRepository interface {
	Create(command *model.Command) error
	Update(command *model.Command) error
	GetList() ([]model.Command, error)
	GetById(id int) (*model.Command, error)
}
type Tracker interface {
	Command() CommandRepository
}
