package util

import (
	gonanoid "github.com/matoous/go-nanoid"
)

func NewPublicID() string {
	return gonanoid.MustGenerate("0123456789abcdefghijklmnopqrstuvwxyz", 16)
}
