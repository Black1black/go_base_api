package entity

import "time"

type User struct {
	ID      int64
	Phone   *int
	Email   *string
	Status  string
	DateReg time.Time
	Name    string
	Photo   *string
	Text    *string
}
