//go:build !windows

package tunnel

// InstallAutoStart is a no-op on non-Windows platforms.
func InstallAutoStart() error { return nil }

// RemoveAutoStart is a no-op on non-Windows platforms.
func RemoveAutoStart() error { return nil }

// IsAutoStartEnabled always returns false on non-Windows platforms.
func IsAutoStartEnabled() bool { return false }
