package accountutil

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"log"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

// copied from https://github.com/marcobeierer/go-accountutils/ on 10.03.2025

// TODO use cryptoutils.RandomString instead of randomStringAsHex
// TODO RandomString uses all printable ASCII and is thus safer than a hex value.

func PBKDF2(password, salt string) string {
	// 210'000 iterations is OWASP recommendation from 2023
	bytes := pbkdf2.Key([]byte(password), []byte(salt), 250000, 64, sha512.New)
	return fmt.Sprintf("%x", bytes)
}

func PasswordHash(plaintextPassword, salt string) string {
	return PBKDF2(plaintextPassword, salt)
}

func RandomSalt() (string, bool) {
	// 128 bit or 32 hex chars is NIST recommendation from 2018
	salt, err := randomStringAsHex(32)
	if err != nil || salt == "" {
		return "", false
	}

	return salt, true
}

func randomStringAsHex(length int64) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Println(err)
		return "", err
	}

	return fmt.Sprintf("%x", bytes), nil
}

// NOTE this is not a complete list of all ascii chars
var asciiRunes = []rune("1234567890QWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnm`!@#$%^*()-_+=[]{}';:|?,./<>&")

func RandomASCIIPassword(length int) (string, bool) {
	return randomPassword(length, asciiRunes)
}

func randomPassword(length int, runes []rune) (string, bool) {
	password := make([]rune, length)

	for i := range password {
		max := big.NewInt(int64(len(runes) - 1))

		bigint, err := rand.Int(rand.Reader, max)
		if err != nil {
			log.Println(err)
			return "", false
		}

		password[i] = runes[bigint.Int64()]
	}

	return string(password), true
}
