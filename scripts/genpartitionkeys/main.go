// Copyright 2026 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command genpartitionkeys generates encrypted libp2p keys and peer IDs for the
// local-soc-partition beekeeper cluster (group-b soft bootnode).
//
//	go run ./scripts/genpartitionkeys/
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/google/uuid"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/scrypt"
)

const (
	password    = "beekeeper"
	scryptN     = 1 << 15
	scryptR     = 8
	scryptP     = 1
	scryptDKLen = 32
)

type encryptedKey struct {
	Address string `json:"address"`
	Crypto  struct {
		Cipher       string `json:"cipher"`
		CipherText   string `json:"ciphertext"`
		CipherParams struct {
			IV string `json:"iv"`
		} `json:"cipherparams"`
		KDF       string `json:"kdf"`
		KDFParams struct {
			N     int    `json:"n"`
			R     int    `json:"r"`
			P     int    `json:"p"`
			DKLen int    `json:"dklen"`
			Salt  string `json:"salt"`
		} `json:"kdfparams"`
		MAC string `json:"mac"`
	} `json:"crypto"`
	Version int    `json:"version"`
	ID      string `json:"id"`
}

func main() {
	for _, name := range []string{"group-b-0", "group-b-1"} {
		priv, err := crypto.GenerateSecp256k1Key()
		if err != nil {
			panic(err)
		}
		keyJSON, err := encryptKey(priv, password)
		if err != nil {
			panic(err)
		}
		pid, err := peerID(priv)
		if err != nil {
			panic(err)
		}
		fmt.Printf("NAME=%s\nPEER=%s\nKEY=%s\n\n", name, pid, keyJSON)
	}
}

func peerID(priv *ecdsa.PrivateKey) (string, error) {
	raw, err := crypto.EncodeSecp256k1PrivateKey(priv)
	if err != nil {
		return "", err
	}
	libKey, err := libp2pcrypto.UnmarshalSecp256k1PrivateKey(raw)
	if err != nil {
		return "", err
	}
	id, err := peer.IDFromPrivateKey(libKey)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func encryptKey(k *ecdsa.PrivateKey, pass string) (string, error) {
	data, err := crypto.EncodeSecp256k1PrivateKey(k)
	if err != nil {
		return "", err
	}
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	derivedKey, err := scrypt.Key([]byte(pass), salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return "", err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(derivedKey[:16])
	if err != nil {
		return "", err
	}
	cipherText := make([]byte, len(data))
	cipher.NewCTR(block, iv).XORKeyStream(cipherText, data)
	mac, err := crypto.LegacyKeccak256(append(derivedKey[16:32], cipherText...))
	if err != nil {
		return "", err
	}
	addr, err := crypto.NewEthereumAddress(k.PublicKey)
	if err != nil {
		return "", err
	}
	ek := encryptedKey{
		Address: hex.EncodeToString(addr),
		Version: 3,
		ID:      uuid.NewString(),
	}
	ek.Crypto.Cipher = "aes-128-ctr"
	ek.Crypto.CipherText = hex.EncodeToString(cipherText)
	ek.Crypto.CipherParams.IV = hex.EncodeToString(iv)
	ek.Crypto.KDF = "scrypt"
	ek.Crypto.KDFParams.N = scryptN
	ek.Crypto.KDFParams.R = scryptR
	ek.Crypto.KDFParams.P = scryptP
	ek.Crypto.KDFParams.DKLen = scryptDKLen
	ek.Crypto.KDFParams.Salt = hex.EncodeToString(salt)
	ek.Crypto.MAC = hex.EncodeToString(mac)
	out, err := json.Marshal(ek)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
