package xweb

import (
	. "gopkg.in/check.v1"
	"log"
	"testing"
	"uuid"
)

func TestAll(t *testing.T) { TestingT(t) }

func TestDebug(t *testing.T) {
	log.Println(uuid.New())
}
