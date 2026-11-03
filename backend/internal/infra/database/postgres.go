package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host, Port, User, Name, SSLMode, SSLRootCert, SearchPath string
	Password                                                 string `json:"-"`
	MaxOpen, MaxIdle                                         int
	StatementTimeout                                         time.Duration
	parameters                                               url.Values
}

// LoadConfig validates runtime settings without exposing credentials in errors.
func LoadConfig() (Config, error) {
	c := Config{Host: os.Getenv("DB_HOST"), Port: os.Getenv("DB_PORT"), User: os.Getenv("DB_USER"), Name: os.Getenv("DB_NAME"),
		SSLMode: os.Getenv("DB_SSLMODE"), SSLRootCert: os.Getenv("DB_SSLROOTCERT"), SearchPath: os.Getenv("DB_SEARCH_PATH"), MaxOpen: 20, MaxIdle: 5, StatementTimeout: 15 * time.Second}
	production := os.Getenv("APP_ENV") == "production"
	if env := os.Getenv("APP_ENV"); env != "" && env != "development" && env != "production" {
		return c, errors.New("invalid APP_ENV")
	}
	connectionURL, alias := os.Getenv("DATABASE_URL"), os.Getenv("Data_Base")
	if connectionURL != "" && alias != "" && connectionURL != alias {
		return c, errors.New("DATABASE_URL and Data_Base contain conflicting configuration; use only one")
	}
	if connectionURL == "" {
		connectionURL = alias
	}
	if connectionURL != "" {
		for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_NAME", "DB_PASSWORD", "DB_PASSWORD_FILE"} {
			if os.Getenv(key) != "" {
				return c, errors.New("configure a database URL or separate DB_* credentials, not both")
			}
		}
		u, err := url.Parse(connectionURL)
		if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.User == nil || u.Fragment != "" {
			return c, errors.New("invalid database URL; expected a complete postgres or postgresql URL")
		}
		parameters, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return c, errors.New("invalid database URL parameters")
		}
		for _, key := range []string{"host", "hostaddr", "port", "user", "password", "dbname", "database"} {
			if parameters.Has(key) {
				return c, errors.New("database URL must not override identity through query parameters")
			}
		}
		for _, values := range parameters {
			if len(values) != 1 {
				return c, errors.New("database URL contains repeated parameters")
			}
		}
		c.Host, c.Port, c.User, c.Name = u.Hostname(), u.Port(), u.User.Username(), strings.TrimPrefix(u.Path, "/")
		c.Password, _ = u.User.Password()
		c.parameters = parameters
		if c.SSLMode == "" {
			mode := parameters.Get("sslmode")
			switch mode {
			case "", "disable", "prefer", "require", "verify-ca", "verify-full":
			default:
				return c, errors.New("invalid database URL TLS mode")
			}
			if production {
				if mode == "disable" {
					return c, errors.New("production database URLs must not disable TLS")
				}
				// Provider URLs commonly use require. Strengthen verification, never weaken it.
				c.SSLMode = "verify-full"
			} else {
				c.SSLMode = mode
			}
		}
		if c.SSLRootCert == "" {
			c.SSLRootCert = parameters.Get("sslrootcert")
		}
		if c.SearchPath == "" {
			c.SearchPath = parameters.Get("search_path")
		}
	}
	if c.Host == "" || c.User == "" || c.Name == "" {
		return c, errors.New("configure DATABASE_URL, Data_Base or the required DB_HOST, DB_USER and DB_NAME fields")
	}
	if c.Port == "" {
		c.Port = "5432"
	}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return c, errors.New("invalid DB_PORT")
	}
	if connectionURL == "" {
		c.Password = os.Getenv("DB_PASSWORD")
	}
	if file := os.Getenv("DB_PASSWORD_FILE"); file != "" {
		if c.Password != "" {
			return c, errors.New("configure only one of DB_PASSWORD and DB_PASSWORD_FILE")
		}
		value, err := os.ReadFile(file)
		if err != nil {
			return c, errors.New("cannot read database password file")
		}
		c.Password = strings.TrimRight(string(value), "\r\n")
	}
	if c.SSLMode == "" {
		if production {
			c.SSLMode = "verify-full"
		} else {
			c.SSLMode = "disable"
		}
	}
	switch c.SSLMode {
	case "disable", "prefer", "require", "verify-ca", "verify-full":
	default:
		return c, errors.New("invalid DB_SSLMODE")
	}
	if production && (c.SSLMode != "verify-full" || c.Password == "") {
		return c, errors.New("production requires DB_SSLMODE=verify-full and a database password")
	}
	if c.SearchPath != "" && !regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`).MatchString(c.SearchPath) {
		return c, errors.New("invalid DB_SEARCH_PATH")
	}
	for name, target := range map[string]*int{"DB_MAX_OPEN_CONNS": &c.MaxOpen, "DB_MAX_IDLE_CONNS": &c.MaxIdle} {
		if value := os.Getenv(name); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return c, fmt.Errorf("invalid %s", name)
			}
			*target = parsed
		}
	}
	if c.MaxOpen < 1 || c.MaxIdle < 0 || c.MaxIdle > c.MaxOpen {
		return c, errors.New("invalid database pool limits")
	}
	if value := os.Getenv("DB_STATEMENT_TIMEOUT"); value != "" {
		c.StatementTimeout, err = time.ParseDuration(value)
		if err != nil || c.StatementTimeout < time.Millisecond {
			return c, errors.New("invalid DB_STATEMENT_TIMEOUT")
		}
	}
	return c, nil
}

func (c Config) dsn() string {
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(c.Host, c.Port), User: url.UserPassword(c.User, c.Password), Path: "/" + c.Name}
	query := make(url.Values, len(c.parameters))
	for key, values := range c.parameters {
		query[key] = append([]string(nil), values...)
	}
	query.Set("sslmode", c.SSLMode)
	query.Set("connect_timeout", "5")
	query.Set("statement_timeout", strconv.FormatInt(c.StatementTimeout.Milliseconds(), 10))
	if c.SSLRootCert != "" {
		query.Set("sslrootcert", c.SSLRootCert)
	}
	if c.SearchPath != "" {
		query.Set("search_path", c.SearchPath)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// Connect creates a bounded pool; callers must close it on shutdown.
func Connect() (*gorm.DB, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(postgres.Open(config.dsn()), &gorm.Config{Logger: safeLogger{}})
	if err != nil {
		return nil, errors.New("database connection failed; verify connectivity and TLS configuration")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, errors.New("database pool initialization failed")
	}
	pool.SetMaxOpenConns(config.MaxOpen)
	pool.SetMaxIdleConns(config.MaxIdle)
	pool.SetConnMaxLifetime(30 * time.Minute)
	pool.SetConnMaxIdleTime(5 * time.Minute)
	return db, nil
}

// safeLogger never logs SQL, arguments, connection strings or dependency errors.
type safeLogger struct{}

func (safeLogger) LogMode(logger.LogLevel) logger.Interface { return safeLogger{} }
func (safeLogger) Info(context.Context, string, ...any)     {}
func (safeLogger) Warn(context.Context, string, ...any)     { slog.Warn("Database warning") }
func (safeLogger) Error(context.Context, string, ...any)    { slog.Error("Database error") }
func (safeLogger) Trace(_ context.Context, begin time.Time, _ func() (string, int64), err error) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("Database operation failed", "durationMs", time.Since(begin).Milliseconds())
	}
}
