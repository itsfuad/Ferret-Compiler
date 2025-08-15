package modules

import (

)

// FerRetDependency represents a dependency entry in fer.ret
type FerRetDependency struct {
	Version string
	Comment string // Optional comment like "used by X"
}
