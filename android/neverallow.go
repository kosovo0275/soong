package android

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/blueprint/proptools"
)

func registerNeverallowMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("neverallow", neverallowMutator).Parallel()
}

var neverallows = []*rule{
	neverallow().
		in("vendor", "device").
		with("vndk.enabled", "true").
		without("vendor", "true").
		because("the VNDK can never contain a library that is device dependent."),
	neverallow().
		with("vndk.enabled", "true").
		without("vendor", "true").
		without("owner", "").
		because("a VNDK module can never have an owner."),
	neverallow().notIn("libcore", "development").with("no_standard_libs", "true"),

	neverallow().
		without("name", "libhidltransport").
		with("product_variables.enforce_vintf_manifest.cflags", "*").
		because("manifest enforcement should be independent of ."),

	neverallow().
		without("name", "libc_bionic_ndk").
		with("product_variables.treble_linker_namespaces.cflags", "*").
		because("nothing should care if linker namespaces are enabled or not"),
}

func neverallowMutator(ctx BottomUpMutatorContext) {
	m, ok := ctx.Module().(Module)
	if !ok {
		return
	}

	dir := ctx.ModuleDir() + "/"
	properties := m.GetProperties()

	for _, n := range neverallows {
		if !n.appliesToPath(dir) {
			continue
		}

		if !n.appliesToProperties(properties) {
			continue
		}

		ctx.ModuleErrorf("violates " + n.String())
	}
}

type ruleProperty struct {
	fields []string // e.x.: Vndk.Enabled
	value  string   // e.x.: true
}

type rule struct {
	reason      string
	paths       []string
	unlessPaths []string
	props       []ruleProperty
	unlessProps []ruleProperty
}

func neverallow() *rule {
	return &rule{}
}
func (r *rule) in(path ...string) *rule {
	r.paths = append(r.paths, cleanPaths(path)...)
	return r
}
func (r *rule) notIn(path ...string) *rule {
	r.unlessPaths = append(r.unlessPaths, cleanPaths(path)...)
	return r
}
func (r *rule) with(properties, value string) *rule {
	r.props = append(r.props, ruleProperty{
		fields: fieldNamesForProperties(properties),
		value:  value,
	})
	return r
}
func (r *rule) without(properties, value string) *rule {
	r.unlessProps = append(r.unlessProps, ruleProperty{
		fields: fieldNamesForProperties(properties),
		value:  value,
	})
	return r
}
func (r *rule) because(reason string) *rule {
	r.reason = reason
	return r
}

func (r *rule) String() string {
	s := "neverallow"
	for _, v := range r.paths {
		s += " dir:" + v + "*"
	}
	for _, v := range r.unlessPaths {
		s += " -dir:" + v + "*"
	}
	for _, v := range r.props {
		s += " " + strings.Join(v.fields, ".") + "=" + v.value
	}
	for _, v := range r.unlessProps {
		s += " -" + strings.Join(v.fields, ".") + "=" + v.value
	}
	if len(r.reason) != 0 {
		s += " which is restricted because " + r.reason
	}
	return s
}

func (r *rule) appliesToPath(dir string) bool {
	includePath := len(r.paths) == 0 || hasAnyPrefix(dir, r.paths)
	excludePath := hasAnyPrefix(dir, r.unlessPaths)
	return includePath && !excludePath
}

func (r *rule) appliesToProperties(properties []interface{}) bool {
	includeProps := hasAllProperties(properties, r.props)
	excludeProps := hasAnyProperty(properties, r.unlessProps)
	return includeProps && !excludeProps
}

func cleanPaths(paths []string) []string {
	res := make([]string, len(paths))
	for i, v := range paths {
		res[i] = filepath.Clean(v) + "/"
	}
	return res
}

func fieldNamesForProperties(propertyNames string) []string {
	names := strings.Split(propertyNames, ".")
	for i, v := range names {
		names[i] = proptools.FieldNameForProperty(v)
	}
	return names
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func hasAnyProperty(properties []interface{}, props []ruleProperty) bool {
	for _, v := range props {
		if hasProperty(properties, v) {
			return true
		}
	}
	return false
}

func hasAllProperties(properties []interface{}, props []ruleProperty) bool {
	for _, v := range props {
		if !hasProperty(properties, v) {
			return false
		}
	}
	return true
}

func hasProperty(properties []interface{}, prop ruleProperty) bool {
	for _, propertyStruct := range properties {
		propertiesValue := reflect.ValueOf(propertyStruct).Elem()
		for _, v := range prop.fields {
			if !propertiesValue.IsValid() {
				break
			}
			propertiesValue = propertiesValue.FieldByName(v)
		}
		if !propertiesValue.IsValid() {
			continue
		}

		check := func(v string) bool {
			return prop.value == "*" || prop.value == v
		}

		if matchValue(propertiesValue, check) {
			return true
		}
	}
	return false
}

func matchValue(value reflect.Value, check func(string) bool) bool {
	if !value.IsValid() {
		return false
	}

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return check("")
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.String:
		return check(value.String())
	case reflect.Bool:
		return check(strconv.FormatBool(value.Bool()))
	case reflect.Int:
		return check(strconv.FormatInt(value.Int(), 10))
	case reflect.Slice:
		slice, ok := value.Interface().([]string)
		if !ok {
			panic("Can only handle slice of string")
		}
		for _, v := range slice {
			if check(v) {
				return true
			}
		}
		return false
	}

	panic("Can't handle type: " + value.Kind().String())
}
