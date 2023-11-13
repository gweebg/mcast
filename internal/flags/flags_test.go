package flags

import (
	"testing"
)

const (
	GET FlagType = 0b1
	SND FlagType = 0b10
	ERR FlagType = 0b100
)

func TestSet(t *testing.T) {
	var flags FlagType = 0b0
	flags.SetFlag(GET)
	if flags != GET {
		t.Fatalf("Expected 0b%08b, but got 0b%08b", GET, flags)
	}
}

func TestUnset(t *testing.T) {
	var flags FlagType = 0b0

	flags.SetFlag(SND)
	flags.SetFlag(GET)
	flags.UnsetFlag(SND)

	if flags != GET {
		t.Fatalf("Expected 0b%08b, but got 0b%08b", GET, flags)
	}
}

func TestCheck(t *testing.T) {
	var flags FlagType = 0b0

	flags.SetFlag(SND)
	flags.SetFlag(GET)
	flags.UnsetFlag(SND)

	if flags.CheckFlag(SND) {
		t.Fatalf("Expected 0b%08b, but got 0b%08b", GET, flags)
	}
}

func TestHasOne(t *testing.T) {

	var flags FlagType = 0b0

	flags.SetFlag(GET)
	//flags.SetFlag(ERR)

	if v := flags.OnlyHasFlag(GET); !v {
		t.Fatalf("Expected %v, but got %v", !v, v)
	}

}
