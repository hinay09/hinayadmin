// Package rsax 登录密码 RSA 加解密工具。
//
// 一次性密钥对流程:
//  1. 前端登录前请求 GET /auth/public-key, 服务端生成 RSA-2048 密钥对,
//     私钥以 keyId 为键存入 Redis(TTL 120s), 公钥(PEM)与 keyId 返回;
//  2. 前端用公钥加密密码(PKCS#1 v1.5), 提交密文与 keyId;
//  3. 服务端按 keyId 原子取出并删除私钥(单次使用), 解密后走原有 bcrypt 校验。
package rsax

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"

	"github.com/gogf/gf/v2/util/guid"
)

// rsaKeyBits 密钥长度。PKCS#1 v1.5 填充下 2048 位最多加密 245 字节, 密码上限 32 字符, 充足。
const rsaKeyBits = 2048

// ErrDecrypt 解密失败(密钥不匹配/密文损坏)的统一错误, 调用方对外不应区分具体原因。
var ErrDecrypt = errors.New("rsa decrypt failed")

// KeyPair 一次性密钥对。
type KeyPair struct {
	KeyId      string // 密钥标识, 同时作为 Redis 私钥键的后缀
	PublicPem  string // 公钥 PEM(SubjectPublicKeyInfo 格式, 前端 jsencrypt 可直接使用)
	PrivatePem string // 私钥 PEM(PKCS#8)
}

// Generate 生成一对一次性密钥。
func Generate() (*KeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, err
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		KeyId: guid.S(),
		PublicPem: string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubDER,
		})),
		PrivatePem: string(pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privDER,
		})),
	}, nil
}

// Decrypt 用私钥 PEM 解密 base64 密文(PKCS#1 v1.5), 返回明文。
func Decrypt(privatePem, base64Cipher string) (string, error) {
	block, _ := pem.Decode([]byte(privatePem))
	if block == nil {
		return "", ErrDecrypt
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", ErrDecrypt
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", ErrDecrypt
	}
	cipherText, err := base64.StdEncoding.DecodeString(base64Cipher)
	if err != nil {
		return "", ErrDecrypt
	}
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, rsaKey, cipherText)
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plain), nil
}
