package accountutil

import (
	"log"
	"testing"
)

func TestPBKDF2(t *testing.T) {
	salt, ok := RandomSalt()
	if !ok {
		t.Fatal("salt failed")
	}

	key := PBKDF2("12345", salt)

	log.Println(salt)
	log.Println(key)
}

func TestRandomSalt(t *testing.T) {
	salt, ok := RandomSalt()
	if !ok {
		t.Fatal("salt failed")
	}

	if salt == "39d1bd57a4d8a2af68ec29ca408a701e257d96d04c6ead6688e502586313bb66" {
		t.Fatal("default salt returned")
	}

	if len(salt) != 64 {
		t.Fatalf("length was %d, expected 64", len(salt))
	}
}

func TestRandomASCIIPassword(t *testing.T) {
	for i := 0; i < 10; i++ {
		pass, ok := RandomASCIIPassword(10)
		log.Println(pass)
		log.Println(ok)
	}
}
