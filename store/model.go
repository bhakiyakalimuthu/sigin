package store

type MethodSignatureEntry struct {
	Id            int    `db:"id"`
	TextSignature string `db:"method_text_signature"`
	HexSignature  string `db:"method_hex_signature"`
}
