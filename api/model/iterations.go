package model

import uuid "github.com/satori/go.uuid"

type Iteration struct {
	ID   *uuid.UUID `jsonapi:"primary,iterations"`
	Name string     `jsonapi:"attr,name"`
}
