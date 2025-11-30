// Package storage provides filesystem utilities for safe and atomic file operations.
//
// This package implements atomic write operations using the standard temp-file + rename
// pattern to ensure data consistency and prevent corruption from partial writes or crashes.
package storage

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicWriteFile writes data to a file atomically using the temp-file + rename pattern.
// This ensures that either the complete file is written or no changes occur, preventing
// partial writes and corruption.
//
// The function creates a temporary file in the same directory as the target, writes the
// data, syncs to disk, and then atomically renames it to the target path. On any error,
// the temporary file is cleaned up and the original file (if it exists) remains unchanged.
//
// Parameters:
//   - path: destination file path
//   - data: bytes to write
//   - perm: file permissions (e.g., 0644)
//
// Returns error if any step fails.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure parent directory: %w", err)
	}

	// Create temp file in same directory as target (required for atomic rename)
	tmpFile, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is on disk before rename
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set correct permissions on temp file
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename - this is the critical operation that makes the write atomic
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Success - prevent cleanup of temp file (it's now the target file)
	tmpFile = nil
	return nil
}

// EnsureDir creates a directory and all necessary parent directories.
// If the directory already exists, it returns nil (no error).
//
// Parameters:
//   - path: directory path to create
//   - perm: directory permissions (e.g., 0755)
//
// Returns error if creation fails.
func EnsureDir(path string, perm os.FileMode) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	// MkdirAll is idempotent - returns nil if dir already exists
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

// FileExists checks if a file exists and is not a directory.
//
// Parameters:
//   - path: file path to check
//
// Returns true if file exists and is a regular file, false otherwise.
func FileExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// DirExists checks if a directory exists.
//
// Parameters:
//   - path: directory path to check
//
// Returns true if directory exists, false otherwise.
func DirExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// SafeRemove removes a file safely, returning nil if the file doesn't exist.
// This function is idempotent - it will not fail if the file is already gone.
//
// Parameters:
//   - path: file path to remove
//
// Returns error only if removal fails (not if file doesn't exist).
func SafeRemove(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	if err := os.Remove(path); err != nil {
		// Ignore "file not found" errors
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}

	return nil
}

// CopyFile copies a file from src to dst safely and efficiently.
// It preserves the source file's permissions and uses buffered I/O for performance.
// The destination file is written atomically using a temp file.
//
// Parameters:
//   - src: source file path
//   - dst: destination file path
//
// Returns error if copy fails at any stage.
func CopyFile(src, dst string) error {
	if src == "" {
		return errors.New("source path cannot be empty")
	}
	if dst == "" {
		return errors.New("destination path cannot be empty")
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure destination directory: %w", err)
	}

	// Create temp file in destination directory
	tmpFile, err := os.CreateTemp(dstDir, ".tmp-"+filepath.Base(dst)+"-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	success := false
	defer func() {
		tmpFile.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Copy data using buffered I/O for efficiency
	bufReader := bufio.NewReader(srcFile)
	bufWriter := bufio.NewWriter(tmpFile)

	if _, err := io.Copy(bufWriter, bufReader); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Flush buffer
	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions to match source
	if err := os.Chmod(tmpPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// ComputeFileHash computes the SHA-256 hash of a file.
// This is useful for integrity verification and deduplication.
//
// Parameters:
//   - path: file path to hash
//
// Returns the SHA-256 hash as a byte slice, or error if computation fails.
func ComputeFileHash(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create hash
	hash := sha256.New()

	// Copy file contents to hash using buffered I/O
	bufReader := bufio.NewReader(file)
	if _, err := io.Copy(hash, bufReader); err != nil {
		return nil, fmt.Errorf("failed to compute hash: %w", err)
	}

	return hash.Sum(nil), nil
}

// GetFileSize returns the size of a file in bytes.
//
// Parameters:
//   - path: file path to check
//
// Returns file size in bytes, or error if stat fails.
func GetFileSize(path string) (int64, error) {
	if path == "" {
		return 0, errors.New("path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return 0, fmt.Errorf("%s is a directory, not a file", path)
	}

	return info.Size(), nil
}
