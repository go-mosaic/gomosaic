package gomosaic

const (
	TransportPkg        = "github.com/go-mosaic/runtime/transport"
	TransportFactoryPkg = "github.com/go-mosaic/runtime/transport/factory"
	SpanPkg             = "github.com/go-mosaic/runtime/span"
	ClientPkg           = "github.com/go-mosaic/runtime/client"
	RuntimePkg          = "github.com/go-mosaic/runtime"
)

// IsTime проверяет, является ли тип time.Time.
func IsTime(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Time"
}

// IsDuration проверяет, является ли тип time.Duration.
func IsDuration(typeInfo *TypeInfo) bool {
	return typeInfo.Package == "time" && typeInfo.Name == "Duration"
}

// HasError проверяет, есть ли в списке переменных ошибка.
func HasError(vars []*VarInfo) (*VarInfo, bool) {
	for _, v := range vars {
		if v.IsError {
			return v, true
		}
	}

	return nil, false
}
