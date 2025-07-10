package daos

import (
	m "github.com/pedrozadotdev/pocketblocks/server/models"
	"github.com/pocketbase/dbx"
)

// PblConnectionQuery returns a new Connection select query.
func (dao *Dao) PblConnectionQuery() *dbx.SelectQuery {
	return dao.ModelQuery(&m.Connection{})
}

// FindPblConnectionById finds a Connection by id.
func (dao *Dao) FindPblConnectionById(id string) (*m.Connection, error) {
	model := &m.Connection{}
	err := dao.PblConnectionQuery().AndWhere(dbx.HashExp{"id": id}).Limit(1).One(model)
	if err != nil {
		return nil, err
	}
	return model, nil
}

// SavePblConnection persists the provided connection.
func (dao *Dao) SavePblConnection(conn *m.Connection) error {
	return dao.Save(conn)
}

// DeletePblConnection deletes the provided connection.
func (dao *Dao) DeletePblConnection(conn *m.Connection) error {
	return dao.Delete(conn)
}
