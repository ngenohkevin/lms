package services

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ngenohkevin/lms/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BackupServiceTestSuite struct {
	suite.Suite
	backupService BackupServiceInterface
	tempDir       string
	db            *database.Database
	ctx           context.Context
}

func TestBackupServiceTestSuite(t *testing.T) {
	suite.Run(t, new(BackupServiceTestSuite))
}

func (s *BackupServiceTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Create temporary directory for test backups
	tempDir, err := os.MkdirTemp("", "lms_backup_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Mock database (in real tests, you'd use a test database)
	s.db = &database.Database{} // This would be properly initialized in real tests

	// Create backup service
	s.backupService = NewBackupService(s.db, s.tempDir, 24*time.Hour)
}

func (s *BackupServiceTestSuite) TearDownSuite() {
	// Clean up temporary directory
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func (s *BackupServiceTestSuite) SetupTest() {
	// Clean backup directory before each test
	files, _ := filepath.Glob(filepath.Join(s.tempDir, "*.sql.gz"))
	for _, file := range files {
		os.Remove(file)
	}
}

func (s *BackupServiceTestSuite) TestCreateMockBackup() {
	// Since we can't actually run pg_dump in tests, we'll create mock backup files
	// to test the backup info generation and file handling logic

	mockBackupContent := `-- PostgreSQL database dump
--
-- Dumped from database version 14.0
-- Dumped by pg_dump version 14.0

SET statement_timeout = 0;
SET lock_timeout = 0;

CREATE TABLE test_table (
    id integer NOT NULL,
    name character varying(100)
);

INSERT INTO test_table VALUES (1, 'test');
`

	// Create a mock backup file
	backupPath := filepath.Join(s.tempDir, "mock_full_20240101_120000_1704110400.sql.gz")
	err := s.createMockGzipFile(backupPath, mockBackupContent)
	s.NoError(err)

	// Verify the mock backup file exists
	_, err = os.Stat(backupPath)
	s.NoError(err)
}

func (s *BackupServiceTestSuite) TestListBackups() {
	// Create some mock backup files
	backups := []struct {
		filename string
		content  string
	}{
		{"full_20240101_120000_1704110400.sql.gz", "-- Full backup content"},
		{"data_only_20240101_130000_1704114000.sql.gz", "-- Data only backup content"},
		{"schema_only_20240101_140000_1704117600.sql.gz", "-- Schema only backup content"},
	}

	for _, backup := range backups {
		backupPath := filepath.Join(s.tempDir, backup.filename)
		err := s.createMockGzipFile(backupPath, backup.content)
		s.NoError(err)
	}

	// List backups
	backupList, err := s.backupService.ListBackups(s.ctx)
	s.NoError(err)
	s.Equal(3, len(backupList))

	// Verify backup types are correctly identified
	backupTypes := make(map[BackupType]bool)
	for _, backup := range backupList {
		backupTypes[backup.Type] = true
		s.True(backup.Size > 0)
		s.NotEmpty(backup.Checksum)
		s.Equal("completed", backup.Status)
		s.True(backup.Compressed)
	}

	s.True(backupTypes[BackupTypeFull])
	s.True(backupTypes[BackupTypeData])
	s.True(backupTypes[BackupTypeSchema])
}

func (s *BackupServiceTestSuite) TestVerifyBackup() {
	// Create a valid mock backup
	validBackupPath := filepath.Join(s.tempDir, "valid_backup.sql.gz")
	err := s.createMockGzipFile(validBackupPath, "-- Valid backup content")
	s.NoError(err)

	// Test verification of valid backup
	verification, err := s.backupService.VerifyBackup(s.ctx, validBackupPath)
	s.NoError(err)
	s.True(verification.IsValid)
	s.True(verification.FileExists)
	s.True(verification.ReadableFile)
	s.True(verification.ChecksumOK)
	s.True(verification.Size > 0)
	s.Empty(verification.Issues)

	// Test verification of non-existent backup
	nonExistentPath := filepath.Join(s.tempDir, "nonexistent.sql.gz")
	verification, err = s.backupService.VerifyBackup(s.ctx, nonExistentPath)
	s.NoError(err)
	s.False(verification.IsValid)
	s.False(verification.FileExists)
	s.Contains(verification.Issues, "backup file does not exist")
}

func (s *BackupServiceTestSuite) TestDeleteBackup() {
	// Create a mock backup to delete
	backupPath := filepath.Join(s.tempDir, "to_delete.sql.gz")
	err := s.createMockGzipFile(backupPath, "-- Backup to delete")
	s.NoError(err)

	// Verify file exists
	_, err = os.Stat(backupPath)
	s.NoError(err)

	// Delete the backup
	err = s.backupService.DeleteBackup(s.ctx, backupPath)
	s.NoError(err)

	// Verify file no longer exists
	_, err = os.Stat(backupPath)
	s.True(os.IsNotExist(err))
}

func (s *BackupServiceTestSuite) TestDeleteBackupSecurityCheck() {
	// Try to delete a file outside the backup directory
	outsidePath := "/tmp/malicious_file.txt"
	err := s.backupService.DeleteBackup(s.ctx, outsidePath)
	s.Error(err)
	s.Contains(err.Error(), "backup path is outside backup directory")
}

func (s *BackupServiceTestSuite) TestCleanupOldBackups() {
	// Create backup service with short retention period
	shortRetentionService := NewBackupService(s.db, s.tempDir, 1*time.Hour)

	// Create old backup (simulate by creating file with old timestamp)
	oldBackupPath := filepath.Join(s.tempDir, "old_full_20240101_120000_1704110400.sql.gz")
	err := s.createMockGzipFile(oldBackupPath, "-- Old backup content")
	s.NoError(err)

	// Set file modification time to 2 hours ago
	oldTime := time.Now().Add(-2 * time.Hour)
	os.Chtimes(oldBackupPath, oldTime, oldTime)

	// Create recent backup
	recentBackupPath := filepath.Join(s.tempDir, "recent_full_20240101_140000_1704117600.sql.gz")
	err = s.createMockGzipFile(recentBackupPath, "-- Recent backup content")
	s.NoError(err)

	// Verify both files exist before cleanup
	_, err = os.Stat(oldBackupPath)
	s.NoError(err)
	_, err = os.Stat(recentBackupPath)
	s.NoError(err)

	// Run cleanup
	err = shortRetentionService.CleanupOldBackups(s.ctx)
	s.NoError(err)

	// Verify old backup is deleted and recent backup remains
	_, err = os.Stat(oldBackupPath)
	s.True(os.IsNotExist(err))
	_, err = os.Stat(recentBackupPath)
	s.NoError(err)
}

func (s *BackupServiceTestSuite) TestGetBackupMetrics() {
	// Create various types of backup files with different sizes
	backups := []struct {
		filename string
		content  string
		status   string
	}{
		{"full_20240101_120000_1704110400.sql.gz", "-- Full backup content with more data to make it larger", "completed"},
		{"full_20240102_120000_1704196800.sql.gz", "-- Another full backup", "completed"},
		{"data_only_20240101_130000_1704114000.sql.gz", "-- Data only backup", "completed"},
		{"schema_only_20240101_140000_1704117600.sql.gz", "-- Schema", "failed"},
	}

	for _, backup := range backups {
		backupPath := filepath.Join(s.tempDir, backup.filename)
		err := s.createMockGzipFile(backupPath, backup.content)
		s.NoError(err)
	}

	// Get backup metrics
	metrics, err := s.backupService.GetBackupMetrics(s.ctx)
	s.NoError(err)

	// Verify metrics
	s.Equal(4, metrics.TotalBackups)
	s.Equal(2, metrics.BackupsByType["full"])
	s.Equal(1, metrics.BackupsByType["data_only"])
	s.Equal(1, metrics.BackupsByType["schema_only"])
	s.True(metrics.TotalSize > 0)
	s.True(metrics.AverageSize > 0)
	s.False(metrics.OldestBackup.IsZero())
	s.False(metrics.NewestBackup.IsZero())
	s.False(metrics.LastBackupTime.IsZero())
}

// Helper methods

func (s *BackupServiceTestSuite) createMockGzipFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create proper gzip content for testing
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	_, err = gzipWriter.Write([]byte(content))
	return err
}

// Unit tests for specific functions

func TestBackupType_String(t *testing.T) {
	tests := []struct {
		backupType BackupType
		expected   string
	}{
		{BackupTypeFull, "full"},
		{BackupTypeData, "data_only"},
		{BackupTypeSchema, "schema_only"},
		{BackupTypeIncremental, "incremental"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, string(test.backupType))
	}
}

func TestNewBackupService(t *testing.T) {
	tempDir := t.TempDir()
	db := &database.Database{}
	retentionPeriod := 48 * time.Hour

	service := NewBackupService(db, tempDir, retentionPeriod)
	assert.NotNil(t, service)

	// Verify backup directory was created
	_, err := os.Stat(tempDir)
	assert.NoError(t, err)
}

func TestBackupService_CalculateChecksum(t *testing.T) {
	// Create temporary file for testing
	tempFile, err := os.CreateTemp("", "checksum_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write test content
	testContent := "test content for checksum calculation"
	_, err = tempFile.WriteString(testContent)
	assert.NoError(t, err)
	tempFile.Close()

	// Create backup service
	bs := &BackupService{}

	// Calculate checksum
	checksum1, err := bs.calculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	// Calculate again to ensure consistency
	checksum2, err := bs.calculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, checksum1, checksum2)

	// Modify file and verify checksum changes
	tempFile2, err := os.OpenFile(tempFile.Name(), os.O_APPEND|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = tempFile2.WriteString(" additional content")
	assert.NoError(t, err)
	tempFile2.Close()

	checksum3, err := bs.calculateChecksum(tempFile.Name())
	assert.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3)
}

// Benchmark tests

func BenchmarkBackupService_CalculateChecksum(b *testing.B) {
	// Create a test file
	tempFile, err := os.CreateTemp("", "benchmark_checksum_*.txt")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write some content
	content := make([]byte, 1024*1024) // 1MB of data
	for i := range content {
		content[i] = byte(i % 256)
	}
	_, err = tempFile.Write(content)
	if err != nil {
		b.Fatal(err)
	}
	tempFile.Close()

	bs := &BackupService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bs.calculateChecksum(tempFile.Name())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBackupService_ListBackups(b *testing.B) {
	// Create temporary directory with mock backups
	tempDir := b.TempDir()

	// Create multiple mock backup files
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("backup_%03d_full_20240101_120000_%d.sql.gz", i, 1704110400+i))
		file, err := os.Create(filename)
		if err != nil {
			b.Fatal(err)
		}
		file.WriteString("-- Mock backup content")
		file.Close()
	}

	bs := NewBackupService(&database.Database{}, tempDir, 24*time.Hour)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bs.ListBackups(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
