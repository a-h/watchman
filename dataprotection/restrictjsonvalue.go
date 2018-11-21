package dataprotection

//RestrictJSONValue checks whether a JSON key is restricted to a given value.
type RestrictJSONValue struct {
	Key   string
	Value interface{}
}

// HasKeyWithUnexpectedValue returns true if the key exists and only contains the
// expected value.
func (rjv RestrictJSONValue) HasKeyWithUnexpectedValue(j map[string]interface{}) bool {
	for k, v := range j {
		if k == rjv.Key {
			if v != rjv.Value {
				return true
			}
		}
		if sub, ok := v.(map[string]interface{}); ok {
			if rjv.HasKeyWithUnexpectedValue(sub) {
				return true
			}
		}
	}
	return false
}
