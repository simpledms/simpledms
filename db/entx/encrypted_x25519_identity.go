package entx

import (
	"database/sql/driver"
	"fmt"
	"log"

	"filippo.io/age"

	"github.com/simpledms/simpledms/encryptor"
)

type EncryptedX25519Identity struct {
	identity *age.X25519Identity
}

func NewEncryptedX25519Identity(identity *age.X25519Identity) EncryptedX25519Identity {
	return EncryptedX25519Identity{identity: identity}
}

func (qq *EncryptedX25519Identity) Identity() *age.X25519Identity {
	return qq.identity
}

func (qq EncryptedX25519Identity) Value() (driver.Value, error) {
	/*
		if qq.identity == nil {
			return nil, nil
		}
	*/

	ciphertext, err := encryptor.Encrypt([]byte(qq.identity.String()))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return ciphertext, nil
}

func (qq *EncryptedX25519Identity) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	val, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan to EncryptedX25519Identity, expected []byte, got %T, value was %v", value, value)
	}

	plaintext, err := encryptor.Decrypt(val)
	if err != nil {
		log.Println(err)
		return err
	}

	identity, err := age.ParseX25519Identity(string(plaintext))
	if err != nil {
		log.Println(err)
		return err
	}

	// TODO does this work or needs indirection?
	qq.identity = identity

	return nil
}
