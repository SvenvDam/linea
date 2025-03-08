package core

type Item[T any] struct {
	Value T
	Err   error
}
