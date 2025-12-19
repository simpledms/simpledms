package recoverx

import (
	"log"
	"runtime/debug"
)

func Recover(name string) {
	if r := recover(); r != nil {
		log.Printf("%v: %s", r, debug.Stack())
		log.Println("trying to recover go routine named:", name)
	}
}
