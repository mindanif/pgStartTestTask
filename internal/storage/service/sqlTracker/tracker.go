package sqlTracker

import (
	"database/sql"
	_ "github.com/lib/pq"
	"pgStartTestTask/internal/storage/service"
)

type Tracker struct {
	db                *sql.DB
	commandRepository *CommandRepository
}

func New(db *sql.DB) *Tracker {
	return &Tracker{
		db: db,
	}
}
func (t *Tracker) Command() service.CommandRepository {
	if t.commandRepository != nil {
		return t.commandRepository
	}
	t.commandRepository = &CommandRepository{
		tracker: t,
	}
	return t.commandRepository
}
