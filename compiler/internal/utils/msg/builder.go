package msg

import (
	"compiler/internal/semantic/stype"

	"fmt"
)

func CastHint(toType stype.Type) string {
	return fmt.Sprintf("You can explicitly cast by writing `as %s` after the expression", toType)
}
