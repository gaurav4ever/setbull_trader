package v2

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogFormatter handles log formatting and file rotation
type LogFormatter struct {
	logDir      string
	logFileName string
	maxSize     int64 // in bytes
	maxFiles    int
}

// NewLogFormatter creates a new log formatter
func NewLogFormatter(logDir, logFileName string, maxSize int64, maxFiles int) *LogFormatter {
	return &LogFormatter{
		logDir:      logDir,
		logFileName: logFileName,
		maxSize:     maxSize,
		maxFiles:    maxFiles,
	}
}

// GetLogFilePath returns the current log file path
func (f *LogFormatter) GetLogFilePath() string {
	return filepath.Join(f.logDir, f.logFileName)
}

// ShouldRotate checks if the log file should be rotated
func (f *LogFormatter) ShouldRotate() bool {
	filePath := f.GetLogFilePath()

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	// Check file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return fileInfo.Size() >= f.maxSize
}

// RotateLogFile rotates the log file
func (f *LogFormatter) RotateLogFile() error {
	filePath := f.GetLogFilePath()

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No file to rotate
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(f.logDir, fmt.Sprintf("%s.%s", f.logFileName, timestamp))

	// Rename current file to backup
	if err := os.Rename(filePath, backupPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Clean up old log files
	if err := f.cleanupOldLogs(); err != nil {
		return fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	return nil
}

// cleanupOldLogs removes old log files beyond the maxFiles limit
func (f *LogFormatter) cleanupOldLogs() error {
	pattern := filepath.Join(f.logDir, f.logFileName+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// If we have more files than maxFiles, remove the oldest ones
	if len(matches) > f.maxFiles {
		// Sort files by modification time (oldest first)
		fileInfos := make([]struct {
			path string
			time time.Time
		}, 0, len(matches))

		for _, match := range matches {
			fileInfo, err := os.Stat(match)
			if err != nil {
				continue
			}
			fileInfos = append(fileInfos, struct {
				path string
				time time.Time
			}{
				path: match,
				time: fileInfo.ModTime(),
			})
		}

		// Sort by modification time (oldest first)
		for i := 0; i < len(fileInfos)-1; i++ {
			for j := i + 1; j < len(fileInfos); j++ {
				if fileInfos[i].time.After(fileInfos[j].time) {
					fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
				}
			}
		}

		// Remove oldest files
		filesToRemove := len(fileInfos) - f.maxFiles
		for i := 0; i < filesToRemove; i++ {
			if err := os.Remove(fileInfos[i].path); err != nil {
				return fmt.Errorf("failed to remove old log file %s: %w", fileInfos[i].path, err)
			}
		}
	}

	return nil
}

// EnsureLogDir ensures the log directory exists
func (f *LogFormatter) EnsureLogDir() error {
	if err := os.MkdirAll(f.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	return nil
}

// GetLogStats returns statistics about log files
func (f *LogFormatter) GetLogStats() (*LogStats, error) {
	stats := &LogStats{
		LogDir:      f.logDir,
		LogFileName: f.logFileName,
		MaxSize:     f.maxSize,
		MaxFiles:    f.maxFiles,
	}

	// Get current log file info
	currentPath := f.GetLogFilePath()
	if fileInfo, err := os.Stat(currentPath); err == nil {
		stats.CurrentFileSize = fileInfo.Size()
		stats.CurrentFileModTime = fileInfo.ModTime()
	}

	// Get backup files info
	pattern := filepath.Join(f.logDir, f.logFileName+".*")
	matches, err := filepath.Glob(pattern)
	if err == nil {
		stats.BackupFileCount = len(matches)
		stats.TotalSize = stats.CurrentFileSize

		for _, match := range matches {
			if fileInfo, err := os.Stat(match); err == nil {
				stats.TotalSize += fileInfo.Size()
			}
		}
	}

	return stats, nil
}

// LogStats contains statistics about log files
type LogStats struct {
	LogDir             string    `json:"log_dir"`
	LogFileName        string    `json:"log_file_name"`
	MaxSize            int64     `json:"max_size"`
	MaxFiles           int       `json:"max_files"`
	CurrentFileSize    int64     `json:"current_file_size"`
	CurrentFileModTime time.Time `json:"current_file_mod_time"`
	BackupFileCount    int       `json:"backup_file_count"`
	TotalSize          int64     `json:"total_size"`
}

// String returns a string representation of LogStats
func (s *LogStats) String() string {
	return fmt.Sprintf("LogDir: %s, CurrentSize: %d bytes, BackupFiles: %d, TotalSize: %d bytes",
		s.LogDir, s.CurrentFileSize, s.BackupFileCount, s.TotalSize)
}
