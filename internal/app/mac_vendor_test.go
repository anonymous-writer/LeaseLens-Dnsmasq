package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOUIDatabaseSupportsIEEEAndArpScanFormats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oui.txt")
	content := "F4F5D8   (hex)               Apple, Inc.\n" +
		"00:00:0C Cisco Systems, Inc.\n" +
		"80691A (base 16) Belkin International Inc.\n" +
		"# comment\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readOUIDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["F4F5D8"] != "Apple, Inc." {
		t.Fatalf("Apple vendor = %q", got["F4F5D8"])
	}
	if got["00000C"] != "Cisco Systems, Inc." {
		t.Fatalf("Cisco vendor = %q", got["00000C"])
	}
	if got["80691A"] != "Belkin International Inc." {
		t.Fatalf("Belkin vendor = %q, want clean name without database marker", got["80691A"])
	}
}

func TestMacVendorIdentifiesLocallyAdministeredAddress(t *testing.T) {
	vendor, private := macVendor("02:11:22:33:44:55")
	if vendor != "" || !private {
		t.Fatalf("macVendor() = (%q, %v), want empty vendor and private=true", vendor, private)
	}
}

func TestMacVendorRejectsInvalidMAC(t *testing.T) {
	vendor, private := macVendor("not-a-mac")
	if vendor != "" || private {
		t.Fatalf("macVendor() = (%q, %v), want empty vendor and private=false", vendor, private)
	}
}
