package orchestration

import (
	"bytes"
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
	"golang.org/x/crypto/sha3"
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

// Decrypt decrypts the encrypted Swarm key with the given password and returns
// the secp256k1 private key. The key file format matches Ethereum JSON v3.
func (k *EncryptedKey) Decrypt(password string) (*ecdsa.PrivateKey, error) {
	if k == nil {
		return nil, fmt.Errorf("nil encrypted key")
	}
	salt, err := hex.DecodeString(k.Crypto.KDFParams.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	derivedKey, err := scrypt.Key([]byte(password), salt, k.Crypto.KDFParams.N, k.Crypto.KDFParams.R, k.Crypto.KDFParams.P, k.Crypto.KDFParams.DKLen)
	if err != nil {
		return nil, fmt.Errorf("scrypt: %w", err)
	}
	cipherText, err := hex.DecodeString(k.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	wantMAC, err := hex.DecodeString(k.Crypto.MAC)
	if err != nil {
		return nil, fmt.Errorf("decode mac: %w", err)
	}
	// Bee keystore MAC is SHA3-256; Ethereum V3 keyfiles use Keccak-256.
	calculatedMAC := sha3.Sum256(append(derivedKey[16:32], cipherText...))
	if !bytes.Equal(calculatedMAC[:], wantMAC) {
		calculatedMACEth, err := crypto.LegacyKeccak256(append(derivedKey[16:32], cipherText...))
		if err != nil {
			return nil, fmt.Errorf("mac: %w", err)
		}
		if !bytes.Equal(calculatedMACEth, wantMAC) {
			return nil, fmt.Errorf("invalid password or corrupted key")
		}
	}
	iv, err := hex.DecodeString(k.Crypto.CipherParams.IV)
	if err != nil {
		return nil, fmt.Errorf("decode iv: %w", err)
	}
	plain, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return crypto.DecodeSecp256k1PrivateKey(plain)
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
