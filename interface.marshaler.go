package model

/*
Marshaler is the interface implemented by types that can serialize
themselves into a static byte array.
*/
type Marshaler interface {
	MarshalModel() ([]byte, error)
}

/*
Unmarshaler is the interface implemented by Models that can unmarshal a
serialized description of themselves. The input can be assumed to be a valid
encoding of a Model value. UnmarshalModel must copy the data if it wishes to
retain the data after returning.

By convention, to approximate the behavior of similar functionality in other
packges, Unmarshalers implement UnmarshalModel([]byte("null")) as a no-op.
*/
type Unmarshaler interface {
	UnmarshalModel(bytes []byte) error
}
