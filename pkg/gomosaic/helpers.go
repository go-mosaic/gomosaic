package gomosaic

func IsTime(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Time"
}

func IsDuration(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Duration"
}

func HasError(vars []*VarInfo) bool {
	for _, v := range vars {
		if v.IsError {
			return true
		}
	}

	return false
}
