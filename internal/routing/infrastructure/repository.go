package infrastructure

import "sync"

type Repository struct{ mu sync.Mutex }

func NewRepository() *Repository { return &Repository{} }
