package gomosaic

// Пути к пакетам runtime, используемым в сгенерированном коде.
const (
	timePkg             = "time"
	timeTypeName        = "Time"
	durationTypeName    = "Duration"
	TransportPkg        = "github.com/go-mosaic/runtime/transport"
	TransportFactoryPkg = "github.com/go-mosaic/runtime/transport/factory"
	SpanPkg             = "github.com/go-mosaic/runtime/span"
	ClientPkg           = "github.com/go-mosaic/runtime/client"
	RuntimePkg          = "github.com/go-mosaic/runtime"
)

// IsTime сообщает, является ли тип time.Time.
func IsTime(typeInfo *TypeInfo) bool {
	return typeInfo.Package == timePkg && typeInfo.Name == timeTypeName
}

// IsDuration сообщает, является ли тип time.Duration.
func IsDuration(typeInfo *TypeInfo) bool {
	return typeInfo.Package == timePkg && typeInfo.Name == durationTypeName
}

// HasError возвращает первую переменную-ошибку и признак, что она есть.
func HasError(vars []*VarInfo) (*VarInfo, bool) {
	for _, v := range vars {
		if v.IsError {
			return v, true
		}
	}

	return nil, false
}
