package serializer

// Global serialize the given render to the general API response format.
func Global(render interface{}) interface{} {
	return map[string]interface{}{
		"data": render,
	}
}
