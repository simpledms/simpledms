package encryptor

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"filippo.io/age"
)

/*
type Encryptor struct{}

func NewEncryptor() *Encryptor {
	return &Encryptor{}
}

*/

func Encrypt(data []byte) ([]byte, error) {
	if NilableX25519MainIdentity == nil {
		return nil, fmt.Errorf("App not unlocked yet, please try again later.")
	}

	ciphertext := &bytes.Buffer{}
	encryptor, err := age.Encrypt(ciphertext, NilableX25519MainIdentity.Recipient())
	if err != nil {
		log.Println(err)
		return nil, err
	}

	plaintextReader := bytes.NewReader(data)

	_, err = io.Copy(encryptor, plaintextReader)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// TODO close in defer on error? or cleaned up on return anyway?
	err = encryptor.Close()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return ciphertext.Bytes(), nil
}

func Decrypt(data []byte) ([]byte, error) {
	if NilableX25519MainIdentity == nil {
		return nil, fmt.Errorf("App not unlocked yet, please try again later.")
	}

	plaintext := &bytes.Buffer{}
	ciphertextReader := bytes.NewReader(data)

	decryptor, err := age.Decrypt(ciphertextReader, NilableX25519MainIdentity)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	_, err = io.Copy(plaintext, decryptor)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return plaintext.Bytes(), nil
}
