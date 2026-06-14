package security

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/model"
)

func TestAdminCredentialVerifiesOnlyAdminKey(t *testing.T) {
	const adminKey = "cmp_admin_test_key_0123456789abcdef"

	credential, err := NewAdminCredential(adminKey, "test")
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if !VerifyAdminKey(credential, adminKey) {
		t.Fatal("admin key did not verify")
	}
	if credential.Algorithm != strongHashAlgorithm {
		t.Fatalf("credential algorithm = %q", credential.Algorithm)
	}
	if credential.Iterations < 100_000 {
		t.Fatalf("credential iterations = %d", credential.Iterations)
	}
	if VerifyAdminKey(credential, "management-key") {
		t.Fatal("cpa management key should not verify as admin key")
	}
	if strings.Contains(credential.KeyHash, adminKey) || strings.Contains(credential.Salt, adminKey) {
		t.Fatalf("credential contains admin key material: %#v", credential)
	}
}

func TestConfigValueDigestVerifiesOnlyOriginalValue(t *testing.T) {
	digest, err := NewConfigValueDigest("https://cpa.local")
	if err != nil {
		t.Fatalf("create digest: %v", err)
	}
	if digest == nil {
		t.Fatal("digest is nil")
	}
	if digest.Algorithm != strongHashAlgorithm {
		t.Fatalf("digest algorithm = %q", digest.Algorithm)
	}
	if digest.Iterations < 100_000 {
		t.Fatalf("digest iterations = %d", digest.Iterations)
	}
	if !VerifyConfigValueDigest(digest, "https://cpa.local") {
		t.Fatal("digest did not verify original value")
	}
	if VerifyConfigValueDigest(digest, "https://other.local") {
		t.Fatal("digest verified a different value")
	}
	if strings.Contains(digest.Hash, "https://cpa.local") || strings.Contains(digest.Salt, "https://cpa.local") {
		t.Fatalf("digest contains original value: %#v", digest)
	}
}

func TestLegacyAdminCredentialStillVerifiesAndNeedsUpgrade(t *testing.T) {
	const adminKey = "cmp_admin_legacy_key"
	salt := []byte("legacy-admin-salt")
	keyHash, err := hashAdminKey(adminKey, salt, 1)
	if err != nil {
		t.Fatalf("legacy hash: %v", err)
	}
	credential := model.AdminCredential{
		Version:    1,
		Salt:       base64.RawStdEncoding.EncodeToString(salt),
		KeyHash:    base64.RawStdEncoding.EncodeToString(keyHash),
		Iterations: 1,
		Source:     "legacy",
	}

	if !VerifyAdminKey(credential, adminKey) {
		t.Fatal("legacy credential did not verify")
	}
	if !AdminCredentialNeedsUpgrade(credential) {
		t.Fatal("legacy credential should need upgrade")
	}
}

func TestGenerateAdminKeyUsesExpectedPrefixAndEntropyLength(t *testing.T) {
	adminKey, err := GenerateAdminKey()
	if err != nil {
		t.Fatalf("generate admin key: %v", err)
	}
	if !strings.HasPrefix(adminKey, "cmp_admin_") {
		t.Fatalf("admin key = %q", adminKey)
	}
	if got, want := len(strings.TrimPrefix(adminKey, "cmp_admin_")), 43; got != want {
		t.Fatalf("encoded random length = %d, want %d", got, want)
	}
}

func TestProtectorEncryptsAndDecryptsString(t *testing.T) {
	protector, err := NewProtector([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create protector: %v", err)
	}

	encrypted, err := protector.ProtectString("management-key")
	if err != nil {
		t.Fatalf("protect string: %v", err)
	}
	if encrypted == "management-key" || !IsProtected(encrypted) {
		t.Fatalf("encrypted value = %q", encrypted)
	}

	plaintext, err := protector.UnprotectString(encrypted)
	if err != nil {
		t.Fatalf("unprotect string: %v", err)
	}
	if plaintext != "management-key" {
		t.Fatalf("plaintext = %q", plaintext)
	}

	otherProtector, err := NewProtector([]byte("abcdef0123456789abcdef0123456789"))
	if err != nil {
		t.Fatalf("create other protector: %v", err)
	}
	if _, err := otherProtector.UnprotectString(encrypted); err == nil {
		t.Fatal("decrypt with wrong data key succeeded")
	}
}

func TestLoadOrCreateDataKeyCreatesStableRestrictedFile(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "data.key")

	first, created, err := LoadOrCreateDataKey("", keyPath)
	if err != nil {
		t.Fatalf("create data key: %v", err)
	}
	if !created || len(first) != 32 {
		t.Fatalf("created=%v len=%d", created, len(first))
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat data key: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("data key permissions = %o, want 600", info.Mode().Perm())
	}

	second, created, err := LoadOrCreateDataKey("", keyPath)
	if err != nil {
		t.Fatalf("load data key: %v", err)
	}
	if created || string(second) != string(first) {
		t.Fatalf("second created=%v key stable=%v", created, string(second) == string(first))
	}
}
