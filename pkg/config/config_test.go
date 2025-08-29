package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLoad(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	
	// Set test values
	viper.Set("database.driver", "postgres")
	viper.Set("database.host", "testhost")
	viper.Set("database.port", 5433)
	viper.Set("database.database", "testdb")
	viper.Set("database.username", "testuser")
	viper.Set("database.password", "testpass")
	viper.Set("verbose", true)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Database.Driver != "postgres" {
		t.Errorf("Expected driver postgres, got %s", cfg.Database.Driver)
	}

	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected host testhost, got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5433 {
		t.Errorf("Expected port 5433, got %d", cfg.Database.Port)
	}

	if !cfg.Verbose {
		t.Error("Expected verbose to be true")
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	
	err := setDefaults(cfg)
	if err != nil {
		t.Fatalf("setDefaults failed: %v", err)
	}

	if cfg.Database.Driver != "postgres" {
		t.Errorf("Expected default driver postgres, got %s", cfg.Database.Driver)
	}

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default host localhost, got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Expected default port 5432, got %d", cfg.Database.Port)
	}

	if cfg.Migration.Directory != "./migrations" {
		t.Errorf("Expected default migration directory ./migrations, got %s", cfg.Migration.Directory)
	}
}

func TestGetDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "PostgreSQL DSN",
			config: Config{
				Database: DatabaseConfig{
					Driver:   "postgres",
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					Username: "user",
					Password: "pass",
					SSLMode:  "disable",
				},
			},
			expected: "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable",
		},
		{
			name: "MySQL DSN",
			config: Config{
				Database: DatabaseConfig{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     3306,
					Database: "testdb",
					Username: "user",
					Password: "pass",
				},
			},
			expected: "user:pass@tcp(localhost:3306)/testdb",
		},
		{
			name: "SQLite DSN",
			config: Config{
				Database: DatabaseConfig{
					Driver:   "sqlite3",
					Database: "/path/to/db.sqlite",
				},
			},
			expected: "/path/to/db.sqlite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.GetDSN()
			if dsn != tt.expected {
				t.Errorf("Expected DSN %s, got %s", tt.expected, dsn)
			}
		})
	}
}