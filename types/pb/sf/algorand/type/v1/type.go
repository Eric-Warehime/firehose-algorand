package pbalgorand

import (
	"time"
)

func (b *Block) Number() uint64 {
	return b.Header.Rnd
}

func (b *Block) Time() time.Time {
	return time.Unix(0, b.Header.Ts).UTC()
}
