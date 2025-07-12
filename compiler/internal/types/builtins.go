package types

type TYPE_NAME string

const (
	INT8         TYPE_NAME = "i8"
	INT16        TYPE_NAME = "i16"
	INT32        TYPE_NAME = "i32"
	INT64        TYPE_NAME = "i64"
	UINT8        TYPE_NAME = "u8"
	UINT16       TYPE_NAME = "u16"
	UINT32       TYPE_NAME = "u32"
	UINT64       TYPE_NAME = "u64"
	FLOAT32      TYPE_NAME = "f32"
	FLOAT64      TYPE_NAME = "f64"
	STRING       TYPE_NAME = "str"
	BYTE         TYPE_NAME = "byte"
	BOOL         TYPE_NAME = "bool"
	FUNCTION     TYPE_NAME = "fn"
	ARRAY        TYPE_NAME = "array"
	INTERFACE    TYPE_NAME = "interface"
	VOID         TYPE_NAME = "void"
	NULL         TYPE_NAME = "null"
	STRUCT       TYPE_NAME = "struct"
	MODULE       TYPE_NAME = "module"
	UNKNOWN_TYPE TYPE_NAME = "unknown"
)

func IsPrimitiveType(name string) bool {
	switch TYPE_NAME(name) {
	case INT8, INT16, INT32, INT64,
		UINT8, UINT16, UINT32, UINT64,
		FLOAT32, FLOAT64,
		STRING, BYTE, BOOL, VOID, NULL:
		return true
	default:
		return false
	}
}
