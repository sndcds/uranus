package model

import "time"

type Todo struct {
	Id          int        `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	DueDate     *time.Time `json:"due_date"`
	Completed   bool       `json:"completed"`
}
