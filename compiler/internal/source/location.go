package source

import "fmt"

// Location represents a span of source code with start and end positions
type Location struct {
	Start *Position
	End   *Position
}

// NewLocation creates a new Location with the given start and end positions
func NewLocation(start, end *Position) *Location {
	return &Location{
		Start: start,
		End:   end,
	}
}

// Contains checks if the given position is within this location
func (l *Location) Contains(pos *Position) bool {
	if l.Start.Line > pos.Line || (l.Start.Line == pos.Line && l.Start.Column > pos.Column) {
		return false
	}
	if l.End.Line < pos.Line || (l.End.Line == pos.Line && l.End.Column < pos.Column) {
		return false
	}
	return true
}

func (l *Location) String() string {
	if l.Start == nil || l.End == nil {
		return "location(unknown)"
	}

	return fmt.Sprintf("location(%d:%d - %d:%d)", l.Start.Line, l.Start.Column, l.End.Line, l.End.Column)
}
