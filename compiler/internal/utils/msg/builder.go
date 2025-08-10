package msg

import (
	"ferret/internal/semantic/stype"

	"fmt"
)

func CastHint(toType stype.Type) string {
	return fmt.Sprintf("Want to cast😐 ? Write `as %s` after the expression", toType)
}
