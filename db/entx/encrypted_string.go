package entx

import (
	"database/sql/driver"
	"fmt"
	"log"

	"github.com/simpledms/simpledms/encryptor"
)

// TODO name?
// TODO string or struct?
type EncryptedString []byte

func NewEncryptedString(s string) EncryptedString {
	return EncryptedString(s)
}

func (qq EncryptedString) String() string {
	return string(qq)
}

// Format redacts secrets in formatted output, including development SQL logs.
// Call String explicitly when plaintext is required by a service client.
func (qq EncryptedString) Format(state fmt.State, _ rune) {
	if _, err := state.Write([]byte("[REDACTED]")); err != nil {
		log.Println("could not format encrypted string")
	}
}

func (qq EncryptedString) Value() (driver.Value, error) {
	/*
		if qq == nil {
			return nil, nil
		}
		if len(qq) == 0 {
			return nil, nil
		}
	*/

	ciphertext, err := encryptor.Encrypt(qq)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ciphertext, nil
}

func (qq *EncryptedString) Scan(value interface{}) error {
	// Newly added optional secrets are NULL on existing installations.
	if value == nil {
		*qq = nil
		return nil
	}
	val, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan to EncryptedString, expected []byte, got %T, value was %v", value, value)
	}

	plaintext, err := encryptor.Decrypt(val)
	if err != nil {
		log.Println(err)
		return err
	}

	*qq = plaintext
	return nil
}
