package request

import "encoding/json"

// Optional is a JSON field that can be missing, null, or have a value. A plain
// Go field cannot tell "the client did not send it" from "the client sent null",
// and for partial updates (PATCH) that difference matters: a missing field
// means "leave it alone", while null means "clear it".
type Optional[T any] struct {
	Set   bool // the field was present in the JSON, even if it was null
	Value *T   // nil if the field was null
}

// UnmarshalJSON is only called by encoding/json when the field is present.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
