package gomosaic

const (
	TransportPkg = "github.com/go-mosaic/runtime/transport"
	LogPkg       = "github.com/go-mosaic/runtime/log"
	RuntimePkg   = "github.com/go-mosaic/runtime"
)

func IsTime(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Time"
}

func IsDuration(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Duration"
}

func HasError(vars []*VarInfo) (*VarInfo, bool) {
	for _, v := range vars {
		if v.IsError {
			return v, true
		}
	}

	return nil, false
}
