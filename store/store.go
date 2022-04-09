package store

type Store interface {
	Insert(entry []*MethodSignatureEntry) error
}
