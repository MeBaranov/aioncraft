package utility

type TheInt struct {
	Value     int
	NAReasons []string
}

func New(a int, name string) *TheInt {
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
