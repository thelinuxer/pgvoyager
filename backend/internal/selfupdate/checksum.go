package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// VerifySHA256 returns nil iff the file at path hashes to the value recorded
// for assetName in a sha256sum-format sumsContent.
func VerifySHA256(path, assetName, sumsContent string) error {
	want, err := sha256FromSums(sumsContent, assetName)
	if err != nil {
		return err
	}
	got, err := sha256File(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("selfupdate: checksum mismatch for %s (got %s, want %s)", assetName, got, want)
	}
	return nil
}

// sha256FromSums finds the hash for assetName in `<hash>  <name>` lines.
func sha256FromSums(content, assetName string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[len(fields)-1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("selfupdate: no checksum found for %s", assetName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
