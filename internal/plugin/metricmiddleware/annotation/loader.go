package annotation

import (
	"github.com/hashicorp/go-multierror"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
	"github.com/go-mosaic/gomosaic/pkg/option"
)

type MethodOpt struct {
	Iface *IfaceOpt
	Func  *gomosaic.MethodInfo

	// @godoc-title "Пропустить генерацию сбора метрик для метода"
	Skip bool `option:"skip,asFlag"`
}

type IfaceOpt struct {
	NameTypeInfo *gomosaic.NameTypeInfo
	// Аннотации для методов
	Methods []*MethodOpt
}

func Load(module *gomosaic.ModuleInfo, prefix string, types []*gomosaic.NameTypeInfo) (interfaces []*IfaceOpt, errs error) {
	for _, nameTypeInfo := range types {
		if nameTypeInfo.Type.Interface == nil {
			continue
		}

		ifaceOpt := &IfaceOpt{NameTypeInfo: nameTypeInfo}

		err := option.Unmarshal(prefix, nameTypeInfo.Annotations, ifaceOpt)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		for _, m := range nameTypeInfo.Type.Interface.Methods {
			methodOpt := &MethodOpt{Iface: ifaceOpt, Func: m}
			err := option.Unmarshal(prefix, m.Annotations, methodOpt)
			if err != nil {
				errs = multierror.Append(errs, err)
			}

			ifaceOpt.Methods = append(ifaceOpt.Methods, methodOpt)
		}

		interfaces = append(interfaces, ifaceOpt)
	}

	return interfaces, errs
}
