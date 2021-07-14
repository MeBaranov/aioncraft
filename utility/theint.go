package utility

import "github.com/google/martian/v3/log"

type TheInt struct {
	Value     int
	NAReasons []string
}

func NewInt(a int, name string) *TheInt {
	return &TheInt{
		Value:     a,
		NAReasons: []string{name},
	}
}

func (a *TheInt) Plus(b *TheInt) *TheInt {
	return &TheInt{
		Value:     a.Value + b.Value,
		NAReasons: append(append([]string(nil), a.NAReasons...), b.NAReasons...),
	}
}

func (a *TheInt) Mul(b int) *TheInt {
	return &TheInt{
		Value:     a.Value * b,
		NAReasons: append([]string(nil), a.NAReasons...),
	}
}

func (a *TheInt) Div(b int) *TheInt {
	if b == 0 {
		log.Errorf("division by zero")
		return &TheInt{0, []string{""}}
	}
	return &TheInt{
		Value:     a.Value / b,
		NAReasons: append([]string(nil), a.NAReasons...),
	}
}
