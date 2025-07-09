package option

func isIn(v any, params ...string) bool {
	for _, param := range params {
		if v == param {
			return true
		}
	}
	return false
}
