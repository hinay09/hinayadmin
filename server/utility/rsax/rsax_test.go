package rsax

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

// encryptWithPublic 模拟前端行为: 用公钥 PEM 做 PKCS#1 v1.5 加密并 base64。
func encryptWithPublic(t *testing.T, publicPem, plaintext string) string {
	t.Helper()
	block, _ := pem.Decode([]byte(publicPem))
	if block == nil {
		t.Fatal("invalid public pem")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		t.Fatal("not an rsa public key")
	}
	cipher, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(plaintext))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return base64.StdEncoding.EncodeToString(cipher)
}

func TestGenerateDecryptRoundTrip(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if kp.KeyId == "" || kp.PublicPem == "" || kp.PrivatePem == "" {
		t.Fatal("incomplete keypair")
	}
	for _, plain := range []string{"123456", "a-very-long-password-with-32-chars!xy", "中文密码测试"} {
		cipher := encryptWithPublic(t, kp.PublicPem, plain)
		got, derr := Decrypt(kp.PrivatePem, cipher)
		if derr != nil {
			t.Fatalf("Decrypt(%q): %v", plain, derr)
		}
		if got != plain {
			t.Fatalf("round trip mismatch: got %q want %q", got, plain)
		}
	}
}

func TestDecryptRejectsInvalidInput(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	valid := encryptWithPublic(t, kp.PublicPem, "123456")

	// 篡改密文
	tampered := []byte(valid)
	if tampered[10] == 'A' {
		tampered[10] = 'B'
	} else {
		tampered[10] = 'A'
	}
	if _, derr := Decrypt(kp.PrivatePem, string(tampered)); derr == nil {
		t.Fatal("tampered ciphertext should fail")
	}
	// 非法 base64
	if _, derr := Decrypt(kp.PrivatePem, "!!!not-base64!!!"); derr == nil {
		t.Fatal("invalid base64 should fail")
	}
	// 私钥不匹配(另一对密钥的私钥)
	other, _ := Generate()
	if _, derr := Decrypt(other.PrivatePem, valid); derr == nil {
		t.Fatal("mismatched private key should fail")
	}
	// 非法私钥 PEM
	if _, derr := Decrypt("not a pem", valid); derr == nil {
		t.Fatal("invalid private pem should fail")
	}
}

func TestKeyPairUniquePerGenerate(t *testing.T) {
	a, _ := Generate()
	b, _ := Generate()
	if a.KeyId == b.KeyId {
		t.Fatal("keyId should be unique per generation")
	}
	if a.PublicPem == b.PublicPem {
		t.Fatal("public key should differ per generation")
	}
}
