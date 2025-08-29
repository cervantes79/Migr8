package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"migr8/pkg/config"
	"migr8/pkg/database"
)

type BackupManager struct {
	db     *database.DB
	config *config.Config
}

type BackupInfo struct {
	Filename    string
	Path        string
	Size        int64
	CreatedAt   time.Time
	Compressed  bool
	DatabaseName string
}

func NewBackupManager(cfg *config.Config) (*BackupManager, error) {
	db, err := database.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &BackupManager{
		db:     db,
		config: cfg,
	}, nil
}

func (bm *BackupManager) Close() error {
	return bm.db.Close()
}

func (bm *BackupManager) Create() (*BackupInfo, error) {
	if err := os.MkdirAll(bm.config.Backup.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.sql", bm.config.Database.Database, timestamp)
	
	if bm.config.Backup.Compression {
		filename += ".gz"
	}
	
	backupPath := filepath.Join(bm.config.Backup.Directory, filename)

	switch bm.db.Driver {
	case "postgres":
		return bm.createPostgresBackup(backupPath, timestamp)
	case "mysql":
		return bm.createMySQLBackup(backupPath, timestamp)
	case "sqlite3":
		return bm.createSQLiteBackup(backupPath, timestamp)
	default:
		return nil, fmt.Errorf("backup not supported for driver: %s", bm.db.Driver)
	}
}

func (bm *BackupManager) createPostgresBackup(backupPath, timestamp string) (*BackupInfo, error) {
	args := []string{
		"-h", bm.config.Database.Host,
		"-p", fmt.Sprintf("%d", bm.config.Database.Port),
		"-U", bm.config.Database.Username,
		"-d", bm.config.Database.Database,
		"--no-password",
		"--verbose",
		"--clean",
		"--no-acl",
		"--no-owner",
	}

	cmd := exec.Command("pg_dump", args...)
	
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bm.config.Database.Password))
	cmd.Env = env

	return bm.executeBackupCommand(cmd, backupPath, timestamp)
}

func (bm *BackupManager) createMySQLBackup(backupPath, timestamp string) (*BackupInfo, error) {
	args := []string{
		"-h", bm.config.Database.Host,
		"-P", fmt.Sprintf("%d", bm.config.Database.Port),
		"-u", bm.config.Database.Username,
		fmt.Sprintf("-p%s", bm.config.Database.Password),
		"--single-transaction",
		"--routines",
		"--triggers",
		bm.config.Database.Database,
	}

	cmd := exec.Command("mysqldump", args...)
	
	return bm.executeBackupCommand(cmd, backupPath, timestamp)
}

func (bm *BackupManager) createSQLiteBackup(backupPath, timestamp string) (*BackupInfo, error) {
	args := []string{
		bm.config.Database.Database,
		".dump",
	}

	cmd := exec.Command("sqlite3", args...)
	
	return bm.executeBackupCommand(cmd, backupPath, timestamp)
}

func (bm *BackupManager) executeBackupCommand(cmd *exec.Cmd, backupPath, timestamp string) (*BackupInfo, error) {
	var file *os.File
	var writer io.Writer
	var err error

	file, err = os.Create(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	writer = file
	compressed := false

	if bm.config.Backup.Compression {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
		compressed = true
	}

	cmd.Stdout = writer
	
	if err := cmd.Run(); err != nil {
		os.Remove(backupPath)
		return nil, fmt.Errorf("backup command failed: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file info: %w", err)
	}

	return &BackupInfo{
		Filename:     filepath.Base(backupPath),
		Path:         backupPath,
		Size:         fileInfo.Size(),
		CreatedAt:    fileInfo.ModTime(),
		Compressed:   compressed,
		DatabaseName: bm.config.Database.Database,
	}, nil
}

func (bm *BackupManager) List() ([]BackupInfo, error) {
	if _, err := os.Stat(bm.config.Backup.Directory); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	files, err := filepath.Glob(filepath.Join(bm.config.Backup.Directory, "*.sql*"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}

	var backups []BackupInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		filename := filepath.Base(file)
		compressed := strings.HasSuffix(filename, ".gz")
		
		parts := strings.Split(filename, "_")
		dbName := "unknown"
		if len(parts) > 0 {
			dbName = parts[0]
		}

		backup := BackupInfo{
			Filename:     filename,
			Path:         file,
			Size:         info.Size(),
			CreatedAt:    info.ModTime(),
			Compressed:   compressed,
			DatabaseName: dbName,
		}
		
		backups = append(backups, backup)
	}

	return backups, nil
}

func (bm *BackupManager) Restore(backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	switch bm.db.Driver {
	case "postgres":
		return bm.restorePostgresBackup(backupPath)
	case "mysql":
		return bm.restoreMySQLBackup(backupPath)
	case "sqlite3":
		return bm.restoreSQLiteBackup(backupPath)
	default:
		return fmt.Errorf("restore not supported for driver: %s", bm.db.Driver)
	}
}

func (bm *BackupManager) restorePostgresBackup(backupPath string) error {
	args := []string{
		"-h", bm.config.Database.Host,
		"-p", fmt.Sprintf("%d", bm.config.Database.Port),
		"-U", bm.config.Database.Username,
		"-d", bm.config.Database.Database,
		"--no-password",
		"--verbose",
	}

	cmd := exec.Command("psql", args...)
	
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bm.config.Database.Password))
	cmd.Env = env

	return bm.executeRestoreCommand(cmd, backupPath)
}

func (bm *BackupManager) restoreMySQLBackup(backupPath string) error {
	args := []string{
		"-h", bm.config.Database.Host,
		"-P", fmt.Sprintf("%d", bm.config.Database.Port),
		"-u", bm.config.Database.Username,
		fmt.Sprintf("-p%s", bm.config.Database.Password),
		bm.config.Database.Database,
	}

	cmd := exec.Command("mysql", args...)
	
	return bm.executeRestoreCommand(cmd, backupPath)
}

func (bm *BackupManager) restoreSQLiteBackup(backupPath string) error {
	tempFile := backupPath + ".tmp"
	
	if strings.HasSuffix(backupPath, ".gz") {
		if err := bm.decompressFile(backupPath, tempFile); err != nil {
			return err
		}
		defer os.Remove(tempFile)
		backupPath = tempFile
	}

	args := []string{
		bm.config.Database.Database,
		fmt.Sprintf(".read %s", backupPath),
	}

	cmd := exec.Command("sqlite3", args...)
	return cmd.Run()
}

func (bm *BackupManager) executeRestoreCommand(cmd *exec.Cmd, backupPath string) error {
	var reader io.Reader
	
	if strings.HasSuffix(backupPath, ".gz") {
		file, err := os.Open(backupPath)
		if err != nil {
			return fmt.Errorf("failed to open backup file: %w", err)
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		reader = gzReader
	} else {
		file, err := os.Open(backupPath)
		if err != nil {
			return fmt.Errorf("failed to open backup file: %w", err)
		}
		defer file.Close()
		
		reader = file
	}

	cmd.Stdin = reader
	return cmd.Run()
}

func (bm *BackupManager) decompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, gzReader)
	return err
}

func (bm *BackupManager) CleanOld() error {
	backups, err := bm.List()
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -bm.config.Backup.RetentionDays)
	var deletedCount int

	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoffTime) {
			if err := os.Remove(backup.Path); err != nil {
				fmt.Printf("Warning: failed to delete old backup %s: %v\n", backup.Filename, err)
			} else {
				deletedCount++
				fmt.Printf("Deleted old backup: %s\n", backup.Filename)
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("Cleaned up %d old backups\n", deletedCount)
	} else {
		fmt.Println("No old backups to clean up")
	}

	return nil
}