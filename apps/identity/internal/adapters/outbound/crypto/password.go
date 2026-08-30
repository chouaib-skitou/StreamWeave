package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

const (
	maxArgonMemoryKiB  = 256 * 1024
	maxArgonIterations = 10
	maxArgonThreads    = 32
	minArgonKeyLength  = 16
	maxArgonKeyLength  = 64
)

func NewPasswordHasher() PasswordHasher {
	return PasswordHasher{time: 3, memory: 64 * 1024, threads: 2, keyLen: 32}
}

func (h PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, h.time, h.memory, h.threads, h.keyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", h.memory, h.time, h.threads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func (h PasswordHasher) Verify(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	parameters := map[string]uint64{}
	for _, parameter := range strings.Split(parts[3], ",") {
		key, value, ok := strings.Cut(parameter, "=")
		if !ok {
			return false
		}
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return false
		}
		parameters[key] = parsed
	}
	memory, memoryOK := parameters["m"]
	iterations, iterationsOK := parameters["t"]
	threads, threadsOK := parameters["p"]
	if !memoryOK || !iterationsOK || !threadsOK || memory == 0 || memory > maxArgonMemoryKiB || iterations == 0 || iterations > maxArgonIterations || threads == 0 || threads > maxArgonThreads {
		return false
	}
	saltText, hashText := parts[4], parts[5]
	salt, err := base64.RawStdEncoding.DecodeString(saltText)
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(hashText)
	if err != nil {
		return false
	}
	if len(salt) < 8 || len(salt) > 64 || len(expected) < minArgonKeyLength || len(expected) > maxArgonKeyLength {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(threads), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
