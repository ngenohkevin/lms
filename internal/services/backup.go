package services

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ngenohkevin/lms/internal/database"
)

type BackupService struct {
	db              *database.Database
	backupDir       string
	retentionPeriod time.Duration
}

type BackupServiceInterface interface {
	CreateBackup(ctx context.Context, backupType BackupType) (*BackupInfo, error)
	RestoreBackup(ctx context.Context, backupPath string) error
	ListBackups(ctx context.Context) ([]BackupInfo, error)
	DeleteBackup(ctx context.Context, backupPath string) error
	VerifyBackup(ctx context.Context, backupPath string) (*BackupVerification, error)
	CleanupOldBackups(ctx context.Context) error
	ScheduleBackups(ctx context.Context) error
	GetBackupMetrics(ctx context.Context) (*BackupMetrics, error)
}

type BackupType string

const (
	BackupTypeFull         BackupType = "full"
	BackupTypeIncremental  BackupType = "incremental"
	BackupTypeData         BackupType = "data_only"
	BackupTypeSchema       BackupType = "schema_only"
)

type BackupInfo struct {
	ID           string        `json:"id"`
	Type         BackupType    `json:"type"`
	FilePath     string        `json:"file_path"`
	FileName     string        `json:"file_name"`
	Size         int64         `json:"size"`
	Checksum     string        `json:"checksum"`
	CreatedAt    time.Time     `json:"created_at"`
	Duration     time.Duration `json:"duration"`
	Compressed   bool          `json:"compressed"`
	DatabaseName string        `json:"database_name"`
	Status       string        `json:"status"`
}

type BackupVerification struct {
	IsValid      bool          `json:"is_valid"`
	ChecksumOK   bool          `json:"checksum_ok"`
	FileExists   bool          `json:"file_exists"`
	ReadableFile bool          `json:"readable_file"`
	Size         int64         `json:"size"`
	Issues       []string      `json:"issues"`
	VerifiedAt   time.Time     `json:"verified_at"`
	Duration     time.Duration `json:"duration"`
}

func NewBackupService(db *database.Database, backupDir string, retentionPeriod time.Duration) BackupServiceInterface {
	if retentionPeriod == 0 {
		retentionPeriod = 30 * 24 * time.Hour // Default 30 days
	}
	
	// Ensure backup directory exists
	os.MkdirAll(backupDir, 0755)
	
	return &BackupService{
		db:              db,
		backupDir:       backupDir,
		retentionPeriod: retentionPeriod,
	}
}

func (bs *BackupService) CreateBackup(ctx context.Context, backupType BackupType) (*BackupInfo, error) {
	start := time.Now()
	
	// Generate backup ID and filename
	backupID := fmt.Sprintf("%s_%s_%d", backupType, time.Now().Format("20060102_150405"), start.Unix())
	filename := fmt.Sprintf("%s.sql.gz", backupID)
	backupPath := filepath.Join(bs.backupDir, filename)
	
	backupInfo := &BackupInfo{
		ID:           backupID,
		Type:         backupType,
		FilePath:     backupPath,
		FileName:     filename,
		CreatedAt:    start,
		Compressed:   true,
		DatabaseName: "lms_db", // This would come from config
		Status:       "in_progress",
	}
	
	// Create backup based on type
	var err error
	switch backupType {
	case BackupTypeFull:
		err = bs.createFullBackup(ctx, backupPath)
	case BackupTypeData:
		err = bs.createDataOnlyBackup(ctx, backupPath)
	case BackupTypeSchema:
		err = bs.createSchemaOnlyBackup(ctx, backupPath)
	case BackupTypeIncremental:
		err = bs.createIncrementalBackup(ctx, backupPath)
	default:
		return nil, fmt.Errorf("unsupported backup type: %s", backupType)
	}
	
	backupInfo.Duration = time.Since(start)
	
	if err != nil {
		backupInfo.Status = "failed"
		// Clean up failed backup file
		os.Remove(backupPath)
		return backupInfo, fmt.Errorf("backup creation failed: %w", err)
	}
	
	// Get file info and calculate checksum
	if fileInfo, err := os.Stat(backupPath); err == nil {
		backupInfo.Size = fileInfo.Size()
	}
	
	checksum, err := bs.calculateChecksum(backupPath)
	if err != nil {
		backupInfo.Status = "completed_with_warnings"
	} else {
		backupInfo.Checksum = checksum
		backupInfo.Status = "completed"
	}
	
	return backupInfo, nil
}

func (bs *BackupService) createFullBackup(ctx context.Context, backupPath string) error {
	// Use pg_dump to create a full backup
	// This is a simplified version - in production, you'd use proper database connection parameters
	
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--verbose",
		"--clean",
		"--if-exists",
		"--create",
		"--format=custom",
		"--no-password",
		bs.getDatabaseURL(),
	)
	
	// Create output file with gzip compression
	outFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()
	
	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()
	
	cmd.Stdout = gzipWriter
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func (bs *BackupService) createDataOnlyBackup(ctx context.Context, backupPath string) error {
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--data-only",
		"--verbose",
		"--format=custom",
		"--no-password",
		bs.getDatabaseURL(),
	)
	
	return bs.executeBackupCommand(cmd, backupPath)
}

func (bs *BackupService) createSchemaOnlyBackup(ctx context.Context, backupPath string) error {
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--schema-only",
		"--verbose",
		"--format=custom",
		"--no-password",
		bs.getDatabaseURL(),
	)
	
	return bs.executeBackupCommand(cmd, backupPath)
}

func (bs *BackupService) createIncrementalBackup(ctx context.Context, backupPath string) error {
	// Incremental backups would require WAL archiving setup
	// This is a placeholder implementation
	// In practice, you'd use pg_basebackup with WAL files
	
	return fmt.Errorf("incremental backups not yet implemented")
}

func (bs *BackupService) executeBackupCommand(cmd *exec.Cmd, backupPath string) error {
	outFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()
	
	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()
	
	cmd.Stdout = gzipWriter
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func (bs *BackupService) RestoreBackup(ctx context.Context, backupPath string) error {
	// Verify backup exists and is valid
	verification, err := bs.VerifyBackup(ctx, backupPath)
	if err != nil {
		return fmt.Errorf("backup verification failed: %w", err)
	}
	
	if !verification.IsValid {
		return fmt.Errorf("backup is not valid: %v", verification.Issues)
	}
	
	// Open and decompress backup file
	backupFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()
	
	gzipReader, err := gzip.NewReader(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	
	// Use pg_restore to restore the backup
	cmd := exec.CommandContext(ctx, "pg_restore",
		"--verbose",
		"--clean",
		"--if-exists",
		"--no-password",
		"--dbname", bs.getDatabaseURL(),
	)
	
	cmd.Stdin = gzipReader
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func (bs *BackupService) ListBackups(ctx context.Context) ([]BackupInfo, error) {
	var backups []BackupInfo
	
	err := filepath.Walk(bs.backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, ".sql.gz") {
			// Parse backup info from filename
			filename := info.Name()
			backupID := strings.TrimSuffix(filename, ".sql.gz")
			
			// Extract backup type from filename
			parts := strings.Split(backupID, "_")
			var backupType BackupType
			if len(parts) >= 2 {
				// Check for multi-part backup types like "data_only" or "schema_only"
				potentialType := parts[0] + "_" + parts[1]
				switch potentialType {
				case string(BackupTypeData), string(BackupTypeSchema):
					backupType = BackupType(potentialType)
				default:
					// Fall back to single part type
					backupType = BackupType(parts[0])
				}
			} else if len(parts) > 0 {
				backupType = BackupType(parts[0])
			}
			
			backup := BackupInfo{
				ID:           backupID,
				Type:         backupType,
				FilePath:     path,
				FileName:     filename,
				Size:         info.Size(),
				CreatedAt:    info.ModTime(),
				Compressed:   true,
				DatabaseName: "lms_db",
				Status:       "completed",
			}
			
			// Calculate checksum
			if checksum, err := bs.calculateChecksum(path); err == nil {
				backup.Checksum = checksum
			}
			
			backups = append(backups, backup)
		}
		
		return nil
	})
	
	return backups, err
}

func (bs *BackupService) DeleteBackup(ctx context.Context, backupPath string) error {
	// Verify the path is within our backup directory (security check)
	absBackupDir, _ := filepath.Abs(bs.backupDir)
	absBackupPath, _ := filepath.Abs(backupPath)
	
	if !strings.HasPrefix(absBackupPath, absBackupDir) {
		return fmt.Errorf("backup path is outside backup directory")
	}
	
	return os.Remove(backupPath)
}

func (bs *BackupService) VerifyBackup(ctx context.Context, backupPath string) (*BackupVerification, error) {
	start := time.Now()
	verification := &BackupVerification{
		VerifiedAt: start,
		Issues:     []string{},
	}
	
	// Check if file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		verification.FileExists = false
		verification.Issues = append(verification.Issues, "backup file does not exist")
	} else {
		verification.FileExists = true
		
		// Check if file is readable
		file, err := os.Open(backupPath)
		if err != nil {
			verification.ReadableFile = false
			verification.Issues = append(verification.Issues, "backup file is not readable")
		} else {
			verification.ReadableFile = true
			file.Close()
			
			// Get file size
			if fileInfo, err := os.Stat(backupPath); err == nil {
				verification.Size = fileInfo.Size()
			}
			
			// Verify gzip compression
			if err := bs.verifyGzipFile(backupPath); err != nil {
				verification.Issues = append(verification.Issues, fmt.Sprintf("gzip verification failed: %v", err))
			}
			
			// Verify checksum if available
			if checksum, err := bs.calculateChecksum(backupPath); err == nil {
				// In a real implementation, you'd compare against stored checksum
				verification.ChecksumOK = len(checksum) > 0
			}
		}
	}
	
	verification.IsValid = len(verification.Issues) == 0
	verification.Duration = time.Since(start)
	
	return verification, nil
}

func (bs *BackupService) CleanupOldBackups(ctx context.Context) error {
	backups, err := bs.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}
	
	cutoffTime := time.Now().Add(-bs.retentionPeriod)
	deletedCount := 0
	
	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoffTime) {
			if err := bs.DeleteBackup(ctx, backup.FilePath); err != nil {
				return fmt.Errorf("failed to delete old backup %s: %w", backup.FilePath, err)
			}
			deletedCount++
		}
	}
	
	return nil
}

func (bs *BackupService) ScheduleBackups(ctx context.Context) error {
	// This would integrate with a job scheduler like cron
	// For now, this is a placeholder
	return fmt.Errorf("backup scheduling not yet implemented")
}

// Helper methods

func (bs *BackupService) getDatabaseURL() string {
	// In production, this would come from configuration
	// For now, return a placeholder
	return "postgresql://user:password@localhost:5432/lms_db"
}

func (bs *BackupService) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (bs *BackupService) verifyGzipFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	
	// Try to read a small amount to verify the file is valid
	buffer := make([]byte, 1024)
	_, err = gzipReader.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}
	
	return nil
}

// BackupMetrics provides metrics about backup operations
type BackupMetrics struct {
	TotalBackups     int           `json:"total_backups"`
	BackupsByType    map[string]int `json:"backups_by_type"`
	TotalSize        int64         `json:"total_size"`
	AverageSize      int64         `json:"average_size"`
	OldestBackup     time.Time     `json:"oldest_backup"`
	NewestBackup     time.Time     `json:"newest_backup"`
	FailedBackups    int           `json:"failed_backups"`
	LastBackupTime   time.Time     `json:"last_backup_time"`
}

func (bs *BackupService) GetBackupMetrics(ctx context.Context) (*BackupMetrics, error) {
	backups, err := bs.ListBackups(ctx)
	if err != nil {
		return nil, err
	}
	
	metrics := &BackupMetrics{
		BackupsByType: make(map[string]int),
	}
	
	var totalSize int64
	var oldest, newest time.Time
	first := true
	
	for _, backup := range backups {
		metrics.TotalBackups++
		metrics.BackupsByType[string(backup.Type)]++
		totalSize += backup.Size
		
		if first {
			oldest = backup.CreatedAt
			newest = backup.CreatedAt
			first = false
		} else {
			if backup.CreatedAt.Before(oldest) {
				oldest = backup.CreatedAt
			}
			if backup.CreatedAt.After(newest) {
				newest = backup.CreatedAt
			}
		}
		
		if backup.Status == "failed" {
			metrics.FailedBackups++
		}
		
		if backup.CreatedAt.After(metrics.LastBackupTime) {
			metrics.LastBackupTime = backup.CreatedAt
		}
	}
	
	metrics.TotalSize = totalSize
	if metrics.TotalBackups > 0 {
		metrics.AverageSize = totalSize / int64(metrics.TotalBackups)
	}
	metrics.OldestBackup = oldest
	metrics.NewestBackup = newest
	
	return metrics, nil
}