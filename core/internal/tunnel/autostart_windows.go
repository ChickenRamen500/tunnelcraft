//go:build windows

package tunnel

import (
	"os"
	"os/exec"
	"path/filepath"
)

// InstallAutoStart adds tunnelcraftd to Windows startup via the Startup folder.
func InstallAutoStart() error {
	exe, _ := os.Executable()
	if exe == "" {
		return nil
	}

	startupFolder := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	shortcutPath := filepath.Join(startupFolder, "TunnelCraft.lnk")

	// Create a VBS script to create a shortcut
	vbsScript := `
Set WshShell = CreateObject("WScript.Shell")
Set oShortcut = WshShell.CreateShortcut("` + shortcutPath + `")
oShortcut.TargetPath = "` + exe + `"
oShortcut.WorkingDirectory = "` + filepath.Dir(exe) + `"
oShortcut.Description = "TunnelCraft VPN"
oShortcut.Save
`

	tmpVbs := filepath.Join(os.TempDir(), "tunnelcraft-shortcut.vbs")
	if err := os.WriteFile(tmpVbs, []byte(vbsScript), 0644); err != nil {
		return err
	}
	defer os.Remove(tmpVbs)

	cmd := exec.Command("cscript", "/nologo", tmpVbs)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// RemoveAutoStart removes tunnelcraftd from Windows startup.
func RemoveAutoStart() error {
	startupFolder := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	shortcutPath := filepath.Join(startupFolder, "TunnelCraft.lnk")

	if _, err := os.Stat(shortcutPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(shortcutPath)
}

// IsAutoStartEnabled checks if auto-start is configured.
func IsAutoStartEnabled() bool {
	startupFolder := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	shortcutPath := filepath.Join(startupFolder, "TunnelCraft.lnk")
	_, err := os.Stat(shortcutPath)
	return err == nil
}