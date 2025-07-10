package mssql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	_ "github.com/denisenkom/go-mssqldb"
)

// Config holds the connection parameters for SQL Server.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// DSN returns a sqlserver connection string for the config.
func (c Config) DSN() string {
	u := &url.URL{
		Scheme: "sqlserver",
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
	}
	if c.User != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	q := url.Values{}
	if c.Database != "" {
		q.Set("database", c.Database)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Open opens a sql.DB connection to the configured SQL Server instance.
func Open(c Config) (*sql.DB, error) {
	return sql.Open("sqlserver", c.DSN())
}

// ConfigFromEnv reads connection parameters from environment variables.
func ConfigFromEnv() Config {
	port, _ := strconv.Atoi(os.Getenv("MSSQL_PORT"))
	if port == 0 {
		port = 1433
	}
	return Config{
		Host:     os.Getenv("MSSQL_HOST"),
		Port:     port,
		User:     os.Getenv("MSSQL_USER"),
		Password: os.Getenv("MSSQL_PASSWORD"),
		Database: os.Getenv("MSSQL_DATABASE"),
	}
}

// ConfigFromJSON decodes Config from JSON data.
func ConfigFromJSON(data []byte) (Config, error) {
	var c Config
	err := json.Unmarshal(data, &c)
	if c.Port == 0 {
		c.Port = 1433
	}
	return c, err
}
