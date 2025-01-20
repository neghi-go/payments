package models

import "github.com/google/uuid"

type Customer struct {
	ID        uuid.UUID `json:"id" db:"id,index,unique,required"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Email     string    `json:"email" db:"email,index,required,unique"`
}
