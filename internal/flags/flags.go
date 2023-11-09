package flags

type FlagType uint8

func (flags *FlagType) CheckFlag(f FlagType) bool {
	return *flags&f != 0
}

func (flags *FlagType) SetFlag(f FlagType) {
	*flags |= f
}

func (flags *FlagType) UnsetFlag(f FlagType) {
	*flags &= ^f
}

func (flags *FlagType) OnlyHasFlag(f FlagType) bool {
	return *flags == f
}
