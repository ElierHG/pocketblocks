package forms

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/pedrozadotdev/pocketblocks/server/daos"
	"github.com/pedrozadotdev/pocketblocks/server/models"
	"github.com/pedrozadotdev/pocketblocks/server/utils"
	v "github.com/pocketbase/pocketbase/forms/validators"
)

// ConnectionUpsert is a [models.Connection] upsert form.
type ConnectionUpsert struct {
	dao        *daos.Dao
	connection *models.Connection

	Id     string `form:"id" json:"id"`
	Name   string `form:"name" json:"name"`
	Type   string `form:"type" json:"type"`
	Config string `form:"config" json:"config"`
}

// NewConnectionUpsert creates a new ConnectionUpsert form.
func NewConnectionUpsert(dao *daos.Dao, conn *models.Connection) *ConnectionUpsert {
	form := &ConnectionUpsert{dao: dao, connection: conn}
	form.Id = conn.Id
	form.Name = conn.Name
	form.Type = conn.Type
	form.Config = conn.Config
	return form
}

// SetDao replaces the default dao.
func (form *ConnectionUpsert) SetDao(dao *daos.Dao) {
	form.dao = dao
}

// Validate implements validation.Validatable.
func (form *ConnectionUpsert) Validate() error {
	return validation.ValidateStruct(form,
		validation.Field(&form.Id,
			validation.When(
				form.connection.IsNew(),
				validation.Length(utils.DefaultIdLength, utils.DefaultIdLength),
				validation.Match(utils.IdRegex),
				validation.By(v.UniqueId(&form.dao.Dao, form.connection.TableName())),
			).Else(validation.In(form.connection.Id)),
		),
		validation.Field(&form.Name, validation.Required),
		validation.Field(&form.Type, validation.Required),
		validation.Field(&form.Config, validation.Required, is.JSON),
	)
}

// Submit validates the form and upserts the connection.
func (form *ConnectionUpsert) Submit() (*models.Connection, error) {
	if err := form.Validate(); err != nil {
		return nil, err
	}

	if form.connection.IsNew() && form.Id != "" {
		form.connection.MarkAsNew()
		form.connection.SetId(form.Id)
	}

	form.connection.Id = form.Id
	form.connection.Name = form.Name
	form.connection.Type = form.Type
	form.connection.Config = form.Config

	if err := form.dao.SavePblConnection(form.connection); err != nil {
		return nil, err
	}

	return form.connection, nil
}
