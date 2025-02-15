package core

// Result wraps a value with a boolean flag indicating success.
// It is used to represent the outcome of stream operations where
// both the value and success status need to be communicated.
//
// Type Parameters:
//   - T: The type of the wrapped value
//
// Fields:
//   - Value: The actual value being wrapped
//   - Ok: Boolean indicating whether the operation was successful
type Result[T any] struct {
	Value T
	Ok    bool
}
