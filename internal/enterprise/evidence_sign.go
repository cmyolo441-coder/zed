package enterprise

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type EvidenceSignature struct {
	Algorithm string    `json:"algorithm"`
	SignedAt  time.Time `json:"signed_at"`
	PublicKey string    `json:"public_key"`
	Signature string    `json:"signature"`
	File      string    `json:"file"`
}

func GenerateEvidenceKeypair() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil { return "", "", err }
	return base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv), nil
}

func SignEvidenceFile(path, privateKeyB64 string) (EvidenceSignature, error) {
	privBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil { return EvidenceSignature{}, err }
	if len(privBytes) != ed25519.PrivateKeySize { return EvidenceSignature{}, fmt.Errorf("invalid ed25519 private key size") }
	buf, err := os.ReadFile(path)
	if err != nil { return EvidenceSignature{}, err }
	priv := ed25519.PrivateKey(privBytes)
	pub := priv.Public().(ed25519.PublicKey)
	sig := ed25519.Sign(priv, buf)
	return EvidenceSignature{Algorithm: "Ed25519", SignedAt: time.Now().UTC(), PublicKey: base64.StdEncoding.EncodeToString(pub), Signature: base64.StdEncoding.EncodeToString(sig), File: filepath.ToSlash(path)}, nil
}

func VerifyEvidenceFile(path string, sig EvidenceSignature) error {
	pub, err := base64.StdEncoding.DecodeString(sig.PublicKey)
	if err != nil { return err }
	sb, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil { return err }
	buf, err := os.ReadFile(path)
	if err != nil { return err }
	if !ed25519.Verify(ed25519.PublicKey(pub), buf, sb) { return fmt.Errorf("evidence signature verification failed") }
	return nil
}

func WriteEvidenceSignature(path, privateKeyB64 string) (string, error) {
	sig, err := SignEvidenceFile(path, privateKeyB64)
	if err != nil { return "", err }
	out := path + ".sig.json"
	buf, _ := json.MarshalIndent(sig, "", "  ")
	if err := os.WriteFile(out, buf, 0644); err != nil { return "", err }
	return out, nil
}
