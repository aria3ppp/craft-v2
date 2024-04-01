package error

import "go/token"

type Position struct {
	Line   int
	Column int
}

func PositionFromToken(p token.Position) Position {
	return Position{
		Line:   p.Line,
		Column: p.Column,
	}
}
