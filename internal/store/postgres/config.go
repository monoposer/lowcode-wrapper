package postgres

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	DSN      string
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string
}

func (c Config) ResolveDSN() (string, error) {
	if dsn := strings.TrimSpace(c.DSN); dsn != "" {
		return dsn, nil
	}

	host := strings.TrimSpace(c.Host)
	user := strings.TrimSpace(c.Username)
	db := strings.TrimSpace(c.Database)
	if host == "" || user == "" || db == "" {
		return "", fmt.Errorf("postgres requires host, username, database (or set dsn / DATABASE_URL)")
	}

	port := c.Port
	if port == 0 {
		port = 5432
	}
	ssl := strings.TrimSpace(c.SSLMode)
	if ssl == "" {
		ssl = "disable"
	}

	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(user, c.Password),
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/" + db,
	}
	q := u.Query()
	q.Set("sslmode", ssl)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
