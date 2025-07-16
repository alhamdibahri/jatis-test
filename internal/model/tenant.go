package model

import "github.com/google/uuid"

type Tenant struct {
	ID      uuid.UUID
	Workers int
}
