package model

import "time"

type Segment struct {
	Id          int        `json:"id"`
	CommandName string     `json:"command_name"`
	Status      string     `json:"status"`
	Output      string     `json:"output"`
	CreatedAt   *time.Time `json:"created_at"`
}
type Command struct {
	Id        int        `json:"id"`
	Script    string     `json:"script"`
	Status    string     `json:"status"`
	Output    string     `json:"output"`
	CreatedAt *time.Time `json:"created_at"`
}
