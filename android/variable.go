package android

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/blueprint/proptools"
)

func init() {
	PreDepsMutators(func(ctx RegisterMutatorsContext) {
		ctx.BottomUp("variable", variableMutator).Parallel()
	})
}

type variableProperties struct {
	Product_variables struct {
		Platform_sdk_version struct {
			Asflags []string
			Cflags  []string
		}

		Unbundled_build struct {
			Enabled *bool `android:"arch_variant"`
		} `android:"arch_variant"`

		Malloc_not_svelte struct {
			Cflags []string
		}

		Safestack struct {
			Cflags []string `android:"arch_variant"`
		} `android:"arch_variant"`

		Binder32bit struct {
			Cflags []string
		}

		Device_uses_hwc2 struct {
			Cflags []string
		}

		Override_rs_driver struct {
			Cflags []string
		}

		Treble_linker_namespaces struct {
			Cflags []string
		}

		Enforce_vintf_manifest struct {
			Cflags []string
		}

		Debuggable struct {
			Cflags   []string
			Cppflags []string
			Init_rc  []string
		}

		Eng struct {
			Cflags   []string
			Cppflags []string
		}

		Pdk struct {
			Enabled *bool `android:"arch_variant"`
		} `android:"arch_variant"`

		Uml struct {
			Cppflags []string
		}
	} `android:"arch_variant"`
}

var zeroProductVariables variableProperties

type productVariables struct {
	Make_suffix *string `json:",omitempty"`

	BuildId             *string `json:",omitempty"`
	BuildNumberFromFile *string `json:",omitempty"`
	DateFromFile        *string `json:",omitempty"`

	Platform_sdk_version              *int     `json:",omitempty"`
	Platform_sdk_codename             *string  `json:",omitempty"`
	Platform_sdk_final                *bool    `json:",omitempty"`
	Platform_version_active_codenames []string `json:",omitempty"`
	Platform_version_future_codenames []string `json:",omitempty"`
	Platform_vndk_version             *string  `json:",omitempty"`
	Platform_systemsdk_versions       []string `json:",omitempty"`

	DeviceName              *string   `json:",omitempty"`
	DeviceArch              *string   `json:",omitempty"`
	DeviceArchVariant       *string   `json:",omitempty"`
	DeviceCpuVariant        *string   `json:",omitempty"`
	DeviceAbi               *[]string `json:",omitempty"`
	DeviceVndkVersion       *string   `json:",omitempty"`
	DeviceSystemSdkVersions *[]string `json:",omitempty"`

	DeviceSecondaryArch        *string   `json:",omitempty"`
	DeviceSecondaryArchVariant *string   `json:",omitempty"`
	DeviceSecondaryCpuVariant  *string   `json:",omitempty"`
	DeviceSecondaryAbi         *[]string `json:",omitempty"`

	HostArch          *string `json:",omitempty"`
	HostSecondaryArch *string `json:",omitempty"`

	CrossHost              *string `json:",omitempty"`
	CrossHostArch          *string `json:",omitempty"`
	CrossHostSecondaryArch *string `json:",omitempty"`

	ResourceOverlays           *[]string `json:",omitempty"`
	EnforceRROTargets          *[]string `json:",omitempty"`
	EnforceRROExcludedOverlays *[]string `json:",omitempty"`

	AAPTCharacteristics *string   `json:",omitempty"`
	AAPTConfig          *[]string `json:",omitempty"`
	AAPTPreferredConfig *string   `json:",omitempty"`
	AAPTPrebuiltDPI     *[]string `json:",omitempty"`

	DefaultAppCertificate *string `json:",omitempty"`

	AppsDefaultVersionName *string `json:",omitempty"`

	Allow_missing_dependencies *bool `json:",omitempty"`
	Unbundled_build            *bool `json:",omitempty"`
	Malloc_not_svelte          *bool `json:",omitempty"`
	Safestack                  *bool `json:",omitempty"`
	HostStaticBinaries         *bool `json:",omitempty"`
	Binder32bit                *bool `json:",omitempty"`
	UseGoma                    *bool `json:",omitempty"`
	Debuggable                 *bool `json:",omitempty"`
	Eng                        *bool `json:",omitempty"`
	Device_uses_hwc2           *bool `json:",omitempty"`
	Treble_linker_namespaces   *bool `json:",omitempty"`
	Sepolicy_split             *bool `json:",omitempty"`
	Enforce_vintf_manifest     *bool `json:",omitempty"`
	Pdk                        *bool `json:",omitempty"`
	Uml                        *bool `json:",omitempty"`
	MinimizeJavaDebugInfo      *bool `json:",omitempty"`

	IntegerOverflowExcludePaths *[]string `json:",omitempty"`

	EnableCFI       *bool     `json:",omitempty"`
	CFIExcludePaths *[]string `json:",omitempty"`
	CFIIncludePaths *[]string `json:",omitempty"`

	VendorPath  *string `json:",omitempty"`
	OdmPath     *string `json:",omitempty"`
	ProductPath *string `json:",omitempty"`

	UseClangLld *bool `json:",omitempty"`

	ClangTidy  *bool   `json:",omitempty"`
	TidyChecks *string `json:",omitempty"`

	NativeCoverage       *bool     `json:",omitempty"`
	CoveragePaths        *[]string `json:",omitempty"`
	CoverageExcludePaths *[]string `json:",omitempty"`

	DevicePrefer32BitExecutables *bool `json:",omitempty"`
	HostPrefer32BitExecutables   *bool `json:",omitempty"`

	SanitizeHost       []string `json:",omitempty"`
	SanitizeDevice     []string `json:",omitempty"`
	SanitizeDeviceDiag []string `json:",omitempty"`
	SanitizeDeviceArch []string `json:",omitempty"`

	ArtUseReadBarrier *bool `json:",omitempty"`

	BtConfigIncludeDir *string `json:",omitempty"`

	Override_rs_driver *string `json:",omitempty"`

	DeviceKernelHeaders []string `json:",omitempty"`
	DistDir             *string  `json:",omitempty"`

	ExtraVndkVersions []string `json:",omitempty"`

	NamespacesToExport []string `json:",omitempty"`

	PgoAdditionalProfileDirs []string `json:",omitempty"`

	VendorVars map[string]map[string]string `json:",omitempty"`
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func (v *productVariables) SetDefaultConfig() {
	*v = productVariables{
		Platform_sdk_version:              intPtr(27),
		Platform_version_active_codenames: []string{"P"},
		Platform_version_future_codenames: []string{"P"},

		HostArch:          stringPtr("arm64"),
		DeviceName:        stringPtr("generic_arm64"),
		DeviceArch:        stringPtr("arm64"),
		DeviceArchVariant: stringPtr("armv8-a"),
		DeviceCpuVariant:  stringPtr("generic"),
		DeviceAbi:         &[]string{"arm64-v8a"},

		AAPTConfig:          &[]string{"normal", "large", "xlarge", "hdpi", "xhdpi", "xxhdpi"},
		AAPTPreferredConfig: stringPtr("xhdpi"),
		AAPTCharacteristics: stringPtr("nosdcard"),
		AAPTPrebuiltDPI:     &[]string{"xhdpi", "xxhdpi"},

		Malloc_not_svelte: boolPtr(true),
		Safestack:         boolPtr(false),
	}
}

func variableMutator(mctx BottomUpMutatorContext) {
	var module Module
	var ok bool
	if module, ok = mctx.Module().(Module); !ok {
		return
	}

	a := module.base()
	variableValues := reflect.ValueOf(&a.variableProperties.Product_variables).Elem()
	zeroValues := reflect.ValueOf(zeroProductVariables.Product_variables)

	for i := 0; i < variableValues.NumField(); i++ {
		variableValue := variableValues.Field(i)
		zeroValue := zeroValues.Field(i)
		name := variableValues.Type().Field(i).Name
		property := "product_variables." + proptools.PropertyNameForField(name)

		val := reflect.ValueOf(mctx.Config().productVariables).FieldByName(name)
		if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
			continue
		}

		val = val.Elem()

		if val.Kind() == reflect.Bool && val.Bool() == false {
			continue
		}

		if reflect.DeepEqual(variableValue.Interface(), zeroValue.Interface()) {
			continue
		}

		a.setVariableProperties(mctx, property, variableValue, val.Interface())
	}
}

func (a *ModuleBase) setVariableProperties(ctx BottomUpMutatorContext,
	prefix string, productVariablePropertyValue reflect.Value, variableValue interface{}) {

	printfIntoProperties(ctx, prefix, productVariablePropertyValue, variableValue)

	err := proptools.AppendMatchingProperties(a.generalProperties,
		productVariablePropertyValue.Addr().Interface(), nil)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			ctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			panic(err)
		}
	}
}

func printfIntoPropertiesError(ctx BottomUpMutatorContext, prefix string,
	productVariablePropertyValue reflect.Value, i int, err error) {

	field := productVariablePropertyValue.Type().Field(i).Name
	property := prefix + "." + proptools.PropertyNameForField(field)
	ctx.PropertyErrorf(property, "%s", err)
}

func printfIntoProperties(ctx BottomUpMutatorContext, prefix string,
	productVariablePropertyValue reflect.Value, variableValue interface{}) {

	for i := 0; i < productVariablePropertyValue.NumField(); i++ {
		propertyValue := productVariablePropertyValue.Field(i)
		kind := propertyValue.Kind()
		if kind == reflect.Ptr {
			if propertyValue.IsNil() {
				continue
			}
			propertyValue = propertyValue.Elem()
		}
		switch propertyValue.Kind() {
		case reflect.String:
			err := printfIntoProperty(propertyValue, variableValue)
			if err != nil {
				printfIntoPropertiesError(ctx, prefix, productVariablePropertyValue, i, err)
			}
		case reflect.Slice:
			for j := 0; j < propertyValue.Len(); j++ {
				err := printfIntoProperty(propertyValue.Index(j), variableValue)
				if err != nil {
					printfIntoPropertiesError(ctx, prefix, productVariablePropertyValue, i, err)
				}
			}
		case reflect.Bool:
			// Nothing
		case reflect.Struct:
			printfIntoProperties(ctx, prefix, propertyValue, variableValue)
		default:
			panic(fmt.Errorf("unsupported field kind %q", propertyValue.Kind()))
		}
	}
}

func printfIntoProperty(propertyValue reflect.Value, variableValue interface{}) error {
	s := propertyValue.String()

	count := strings.Count(s, "%")
	if count == 0 {
		return nil
	}

	if count > 1 {
		return fmt.Errorf("product variable properties only support a single '%%'")
	}

	if strings.Contains(s, "%d") {
		switch v := variableValue.(type) {
		case int:
			// Nothing
		case bool:
			if v {
				variableValue = 1
			} else {
				variableValue = 0
			}
		default:
			return fmt.Errorf("unsupported type %T for %%d", variableValue)
		}
	} else if strings.Contains(s, "%s") {
		switch variableValue.(type) {
		case string:
			// Nothing
		default:
			return fmt.Errorf("unsupported type %T for %%s", variableValue)
		}
	} else {
		return fmt.Errorf("unsupported %% in product variable property")
	}

	propertyValue.Set(reflect.ValueOf(fmt.Sprintf(s, variableValue)))

	return nil
}
