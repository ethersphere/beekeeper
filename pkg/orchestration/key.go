package orchestration

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
	"golang.org/x/crypto/scrypt"
	"gopkg.in/yaml.v3"
)

const (
	keyHeaderKDF = "scrypt"
	keyVersion   = 3

	scryptN     = 1 << 15
	scryptR     = 8
	scryptP     = 1
	scryptDKLen = 32
)

func NewEncryptedKey(password string) (*EncryptedKey, error) {
	key, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return nil, err
	}

	encrypted, err := encryptKey(key, password)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

// This format is compatible with Ethereum JSON v3 key file format.
type EncryptedKey struct {
	Address string    `json:"address"`
	Crypto  keyCripto `json:"crypto"`
	Version int       `json:"version"`
	ID      string    `json:"id"`
}

func (k *EncryptedKey) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("expected swarm-key as a JSON string but got something else: %w", err)
	}

	if err := json.Unmarshal([]byte(raw), k); err != nil {
		return fmt.Errorf("failed to parse EncryptedKey from JSON: %w", err)
	}

	return nil
}

func (k *EncryptedKey) StringJSON() (string, error) {
	data, err := json.Marshal(k)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type keyCripto struct {
	Cipher       string       `json:"cipher"`
	CipherText   string       `json:"ciphertext"`
	CipherParams cipherParams `json:"cipherparams"`
	KDF          string       `json:"kdf"`
	KDFParams    kdfParams    `json:"kdfparams"`
	MAC          string       `json:"mac"`
}

type cipherParams struct {
	IV string `json:"iv"`
}

type kdfParams struct {
	N     int    `json:"n"`
	R     int    `json:"r"`
	P     int    `json:"p"`
	DKLen int    `json:"dklen"`
	Salt  string `json:"salt"`
}

func encryptKey(k *ecdsa.PrivateKey, password string) (*EncryptedKey, error) {
	data, err := crypto.EncodeSecp256k1PrivateKey(k)
	if err != nil {
		return nil, err
	}
	kc, err := encryptData(data, []byte(password))
	if err != nil {
		return nil, err
	}
	addr, err := crypto.NewEthereumAddress(k.PublicKey)
	if err != nil {
		return nil, err
	}
	return &EncryptedKey{
		Address: hex.EncodeToString(addr),
		Crypto:  *kc,
		Version: keyVersion,
		ID:      uuid.NewString(),
	}, nil
}

func encryptData(data, password []byte) (*keyCripto, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("read random data: %w", err)
	}
	derivedKey, err := scrypt.Key(password, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("read random data: %w", err)
	}
	cipherText, err := aesCTRXOR(encryptKey, data, iv)
	if err != nil {
		return nil, err
	}
	mac, err := crypto.LegacyKeccak256(append(derivedKey[16:32], cipherText...))
	if err != nil {
		return nil, err
	}

	return &keyCripto{
		Cipher:     "aes-128-ctr",
		CipherText: hex.EncodeToString(cipherText),
		CipherParams: cipherParams{
			IV: hex.EncodeToString(iv),
		},
		KDF: keyHeaderKDF,
		KDFParams: kdfParams{
			N:     scryptN,
			R:     scryptR,
			P:     scryptP,
			DKLen: scryptDKLen,
			Salt:  hex.EncodeToString(salt),
		},
		MAC: hex.EncodeToString(mac[:]),
	}, nil
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, nil
}
