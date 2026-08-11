package gomosaic

const (
	TransportPkg        = "github.com/go-mosaic/runtime/transport"
	TransportFactoryPkg = "github.com/go-mosaic/runtime/transport/factory"
	SpanPkg             = "github.com/go-mosaic/runtime/span"
	ClientPkg           = "github.com/go-mosaic/runtime/client"
	RuntimePkg          = "github.com/go-mosaic/runtime"
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
