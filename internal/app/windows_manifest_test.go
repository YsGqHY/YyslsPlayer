package app

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

type windowsManifest struct {
	TrustInfo struct {
		Security struct {
			RequestedPrivileges struct {
				RequestedExecutionLevel struct {
					Level    string `xml:"level,attr"`
					UIAccess string `xml:"uiAccess,attr"`
				} `xml:"requestedExecutionLevel"`
			} `xml:"requestedPrivileges"`
		} `xml:"security"`
	} `xml:"trustInfo"`
}

func TestWindowsManifestRequiresAdministrator(t *testing.T) {
	path := filepath.Join("..", "..", "build", "windows", "wails.exe.manifest")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	var manifest windowsManifest
	if err := xml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	level := manifest.TrustInfo.Security.RequestedPrivileges.RequestedExecutionLevel.Level
	if level != "requireAdministrator" {
		t.Fatalf("requestedExecutionLevel = %q, want requireAdministrator", level)
	}
	uiAccess := manifest.TrustInfo.Security.RequestedPrivileges.RequestedExecutionLevel.UIAccess
	if uiAccess != "false" {
		t.Fatalf("uiAccess = %q, want false", uiAccess)
	}
}
