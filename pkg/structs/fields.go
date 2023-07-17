package structs

import "github.com/oleiade/reflections"

// GetField returns the value of the provided obj field. obj can whether be a structure or pointer to structure.
func GetField(obj any, name string) any {
	v, err := reflections.GetField(obj, name)
	if err != nil {
		panic(err)
	}

	return v
}

// SetField sets the provided obj field with provided value.
// obj param has to be a pointer to a struct, otherwise it will soundly fail.
// Provided value type should match with the struct field you're trying to set.
func SetField(obj any, name string, value any) {
	if err := reflections.SetField(obj, name, value); err != nil {
		panic(err)
	}
}
