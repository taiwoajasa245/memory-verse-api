package util

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// ---------- BCRYPT (easy, compatible) ----------

// BcryptCost is the bcrypt cost parameter. 12 is a good default for production.
// You can increase to 13 or 14 if your environment can handle it.
const BcryptCost = 12

// HashPasswordBcrypt returns a bcrypt hash of the given plaintext password.
// Store the returned string in your DB (it already includes salt).
func HashPasswordBcrypt(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is empty")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePasswordBcrypt returns nil if the plaintext password matches the bcrypt hash.
func ComparePasswordBcrypt(hashedPassword, password string) error {
	if hashedPassword == "" || password == "" {
		return bcrypt.ErrMismatchedHashAndPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}





// ---------- ARGON2id (recommended for new systems) ----------
// We encode the final value as:
// argon2id$v=19$m=<memory>,t=<time>,p=<threads>$<base64_salt>$<base64_hash>

// Default Argon2 params. Tune these according to your hardware.
var (
	ArgonTime    uint32 = 1               // number of iterations
	ArgonMemory  uint32 = 64 * 1024       // 64 MB
	ArgonThreads uint8  = 4               // parallelism
	ArgonKeyLen  uint32 = 32              // output length in bytes
	SaltLen              = 16             // 16 bytes salt
)

// HashPasswordArgon2 hashes a password using Argon2id and returns an encoded string
// containing parameters, salt and hash. Safe to store directly in DB.
func HashPasswordArgon2(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is empty")
	}

	salt, err := generateRandomBytes(SaltLen)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		ArgonMemory, ArgonTime, ArgonThreads, b64Salt, b64Hash)

	return encoded, nil
}

// ComparePasswordArgon2 compares an encoded argon2id hash with a plaintext password.
// Returns nil on success, or an error on failure.
func ComparePasswordArgon2(encodedHash, password string) error {
	if encodedHash == "" || password == "" {
		return errors.New("invalid input")
	}

	// format: argon2id$v=19$m=<memory>,t=<time>,p=<threads>$<base64_salt>$<base64_hash>
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 {
		return errors.New("invalid hash format")
	}

	paramsPart := parts[2] // e.g. m=65536,t=1,p=4
	saltB64 := parts[3]
	hashB64 := parts[4]

	var memory uint32
	var timeParam uint32
	var threads uint8
	_, err := fmt.Sscanf(paramsPart, "m=%d,t=%d,p=%d", &memory, &timeParam, &threads)
	if err != nil {
		// fallback parse: some encoders separate with commas; try manual parse
		// but for simplicity we return an error here.
		return fmt.Errorf("failed to parse argon2 params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return err
	}
	hash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return err
	}

	derived := argon2.IDKey([]byte(password), salt, timeParam, memory, threads, uint32(len(hash)))

	// constant-time comparison
	if subtle.ConstantTimeCompare(hash, derived) == 1 {
		return nil
	}
	return errors.New("password mismatch")
}

// ---------- helpers ----------

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
