package sshutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gliderlabs/ssh"
)

func EnsureHostKey(path string) (ssh.Option, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := generateHostKey(path); err != nil {
			return nil, fmt.Errorf("generate host key: %w", err)
		}
	}
	return ssh.HostKeyFile(path), nil
}

func generateHostKey(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}
