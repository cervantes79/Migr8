package seed

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"migr8/pkg/config"
	"migr8/pkg/database"
)

type Seeder struct {
	db     *database.DB
	config *config.Config
}

type SeedFile struct {
	Name     string            `yaml:"name"`
	Table    string            `yaml:"table"`
	Truncate bool              `yaml:"truncate"`
	Data     []map[string]interface{} `yaml:"data"`
}

type CSVSeedFile struct {
	Filename string
	Table    string
	Truncate bool
}

func NewSeeder(cfg *config.Config) (*Seeder, error) {
	db, err := database.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &Seeder{
		db:     db,
		config: cfg,
	}, nil
}

func (s *Seeder) Close() error {
	return s.db.Close()
}

func (s *Seeder) Run() error {
	if _, err := os.Stat(s.config.Seed.Directory); os.IsNotExist(err) {
		fmt.Printf("Seed directory does not exist: %s\n", s.config.Seed.Directory)
		return nil
	}

	yamlFiles, err := s.loadYAMLSeeds()
	if err != nil {
		return fmt.Errorf("failed to load YAML seeds: %w", err)
	}

	csvFiles, err := s.loadCSVSeeds()
	if err != nil {
		return fmt.Errorf("failed to load CSV seeds: %w", err)
	}

	if len(yamlFiles) == 0 && len(csvFiles) == 0 {
		fmt.Println("No seed files found.")
		return nil
	}

	fmt.Printf("Processing %d YAML seed files and %d CSV seed files...\n", len(yamlFiles), len(csvFiles))

	for _, seedFile := range yamlFiles {
		if err := s.processYAMLSeed(seedFile); err != nil {
			return fmt.Errorf("failed to process YAML seed %s: %w", seedFile.Name, err)
		}
		fmt.Printf("Processed YAML seed: %s (%s)\n", seedFile.Name, seedFile.Table)
	}

	for _, csvFile := range csvFiles {
		if err := s.processCSVSeed(csvFile); err != nil {
			return fmt.Errorf("failed to process CSV seed %s: %w", csvFile.Filename, err)
		}
		fmt.Printf("Processed CSV seed: %s (%s)\n", csvFile.Filename, csvFile.Table)
	}

	fmt.Println("All seeds processed successfully!")
	return nil
}

func (s *Seeder) loadYAMLSeeds() ([]SeedFile, error) {
	files, err := filepath.Glob(filepath.Join(s.config.Seed.Directory, "*.yml"))
	if err != nil {
		return nil, err
	}
	
	yamlFiles, err := filepath.Glob(filepath.Join(s.config.Seed.Directory, "*.yaml"))
	if err != nil {
		return nil, err
	}
	
	files = append(files, yamlFiles...)

	var seedFiles []SeedFile
	seedNameRegex := regexp.MustCompile(`^(\d+)_(.+)\.(yml|yaml)$`)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read seed file %s: %w", file, err)
		}

		var seedFile SeedFile
		if err := yaml.Unmarshal(content, &seedFile); err != nil {
			return nil, fmt.Errorf("failed to parse YAML seed file %s: %w", file, err)
		}

		basename := filepath.Base(file)
		matches := seedNameRegex.FindStringSubmatch(basename)
		if len(matches) == 4 {
			seedFile.Name = matches[2]
		} else {
			seedFile.Name = strings.TrimSuffix(basename, filepath.Ext(basename))
		}

		seedFiles = append(seedFiles, seedFile)
	}

	sort.Slice(seedFiles, func(i, j int) bool {
		return seedFiles[i].Name < seedFiles[j].Name
	})

	return seedFiles, nil
}

func (s *Seeder) loadCSVSeeds() ([]CSVSeedFile, error) {
	files, err := filepath.Glob(filepath.Join(s.config.Seed.Directory, "*.csv"))
	if err != nil {
		return nil, err
	}

	var csvFiles []CSVSeedFile
	csvNameRegex := regexp.MustCompile(`^(\d+_)?(.+)\.csv$`)

	for _, file := range files {
		basename := filepath.Base(file)
		matches := csvNameRegex.FindStringSubmatch(basename)
		
		var tableName string
		if len(matches) >= 3 {
			tableName = matches[2]
		} else {
			tableName = strings.TrimSuffix(basename, ".csv")
		}

		csvFile := CSVSeedFile{
			Filename: basename,
			Table:    tableName,
			Truncate: true,
		}

		csvFiles = append(csvFiles, csvFile)
	}

	sort.Slice(csvFiles, func(i, j int) bool {
		return csvFiles[i].Filename < csvFiles[j].Filename
	})

	return csvFiles, nil
}

func (s *Seeder) processYAMLSeed(seedFile SeedFile) error {
	if seedFile.Table == "" {
		return fmt.Errorf("table name is required for seed %s", seedFile.Name)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if seedFile.Truncate {
		if err := s.truncateTable(tx, seedFile.Table); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", seedFile.Table, err)
		}
	}

	for _, row := range seedFile.Data {
		if err := s.insertRow(tx, seedFile.Table, row); err != nil {
			return fmt.Errorf("failed to insert row into %s: %w", seedFile.Table, err)
		}
	}

	return tx.Commit()
}

func (s *Seeder) processCSVSeed(csvFile CSVSeedFile) error {
	filePath := filepath.Join(s.config.Seed.Directory, csvFile.Filename)
	
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("CSV file is empty")
	}

	headers := records[0]
	dataRows := records[1:]

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if csvFile.Truncate {
		if err := s.truncateTable(tx, csvFile.Table); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", csvFile.Table, err)
		}
	}

	for _, row := range dataRows {
		if len(row) != len(headers) {
			continue
		}

		rowData := make(map[string]interface{})
		for i, value := range row {
			rowData[headers[i]] = value
		}

		if err := s.insertRow(tx, csvFile.Table, rowData); err != nil {
			return fmt.Errorf("failed to insert CSV row into %s: %w", csvFile.Table, err)
		}
	}

	return tx.Commit()
}

func (s *Seeder) truncateTable(tx *sql.Tx, tableName string) error {
	var query string
	
	switch s.db.Driver {
	case "postgres":
		query = fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)
	case "mysql":
		query = fmt.Sprintf("TRUNCATE TABLE %s", tableName)
	case "sqlite3":
		query = fmt.Sprintf("DELETE FROM %s", tableName)
	default:
		return fmt.Errorf("truncate not supported for driver: %s", s.db.Driver)
	}

	_, err := tx.Exec(query)
	return err
}

func (s *Seeder) insertRow(tx *sql.Tx, tableName string, data map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	var columns []string
	var placeholders []string
	var values []interface{}

	i := 1
	for column, value := range data {
		columns = append(columns, column)
		values = append(values, value)
		
		if s.db.Driver == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		} else {
			placeholders = append(placeholders, "?")
		}
		i++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	_, err := tx.Exec(query, values...)
	return err
}

func (s *Seeder) GenerateTemplate(tableName string) error {
	if err := os.MkdirAll(s.config.Seed.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create seed directory: %w", err)
	}

	template := SeedFile{
		Name:     tableName,
		Table:    tableName,
		Truncate: true,
		Data: []map[string]interface{}{
			{
				"id":         1,
				"name":       "Sample Name",
				"email":      "sample@example.com",
				"created_at": "2023-01-01T00:00:00Z",
			},
			{
				"id":         2,
				"name":       "Another Sample",
				"email":      "another@example.com", 
				"created_at": "2023-01-02T00:00:00Z",
			},
		},
	}

	yamlData, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	filename := filepath.Join(s.config.Seed.Directory, fmt.Sprintf("%s.yml", tableName))
	
	header := fmt.Sprintf(`# Seed file for %s table
# Generated on: %s
# 
# This file contains sample data for seeding the %s table.
# Modify the data below to match your table structure.
#
# Format:
# - name: Human readable name for this seed
# - table: Target table name  
# - truncate: Whether to truncate table before seeding (true/false)
# - data: Array of records to insert
#

`, tableName, time.Now().Format("2006-01-02 15:04:05"), tableName)

	content := header + string(yamlData)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	fmt.Printf("Generated seed template: %s\n", filename)
	return nil
}