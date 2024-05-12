package sqlTracker

import (
	"database/sql"
	"pgStartTestTask/internal/storage/model"
	"pgStartTestTask/internal/storage/service"
)

type CommandRepository struct {
	tracker *Tracker
}

func (r *CommandRepository) Update(command *model.Command) error {
	// Update command in the database
	_, err := r.tracker.db.Exec("UPDATE command SET script=$1, status=$2, output=$3 WHERE id=$4",
		command.Script, command.Status, command.Output, command.Id)
	if err != nil {
		return err
	}
	return nil
}
func (r *CommandRepository) Create(command *model.Command) error {
	// Insert command into database
	return r.tracker.db.QueryRow("INSERT INTO command(script, status, output) VALUES($1, $2, $3) RETURNING id, created_at",
		command.Script, command.Status, command.Output).Scan(&command.Id, &command.CreatedAt)
}

func (r *CommandRepository) GetList() ([]model.Command, error) {
	commands := make([]model.Command, 0)
	rows, err := r.tracker.db.Query(
		"SELECT * FROM command",
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrorRecordNotFound
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var command model.Command
		if err := rows.Scan(&command.Id, &command.Script, &command.Status, &command.Output, &command.CreatedAt); err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return commands, nil
}
func (r *CommandRepository) GetById(id int) (*model.Command, error) {
	row := r.tracker.db.QueryRow(
		"SELECT * FROM command WHERE id=$1",
		id,
	)
	var command model.Command
	err := row.Scan(&command.Id, &command.Script, &command.Status, &command.Output, &command.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &command, nil
}
