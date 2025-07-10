package models

import (
	m "github.com/pocketbase/pocketbase/models"
)

var _ m.Model = (*Connection)(nil)

// Connection represents an external datasource connection like MSSQL.
type Connection struct {
	m.BaseModel

	Name   string `db:"name" json:"name"`
	Type   string `db:"type" json:"type"`
	Config string `db:"config" json:"config"`
}

// TableName returns the database table name.
func (m *Connection) TableName() string {
	return "_pbl_connections"
}
