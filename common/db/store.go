package db

type Store interface {
	GetDB(id []byte) (DB, error)
}

func GetStore(name string) (Store, error) {
	return nil, nil
}