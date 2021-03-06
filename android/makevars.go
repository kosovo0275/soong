package android

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/google/blueprint/proptools"
)

func init() {
	RegisterMakeVarsProvider(pctx, androidMakeVarsProvider)
}

func androidMakeVarsProvider(ctx MakeVarsContext) {
	ctx.Strict("MIN_SUPPORTED_SDK_VERSION", strconv.Itoa(ctx.Config().MinSupportedSdkVersion()))
}

type MakeVarsContext interface {
	Config() Config
	DeviceConfig() DeviceConfig
	SingletonContext() SingletonContext
	Strict(name, ninjaStr string)
	Check(name, ninjaStr string)
	StrictSorted(name, ninjaStr string)
	CheckSorted(name, ninjaStr string)
	Eval(ninjaStr string) (string, error)
	StrictRaw(name, value string)
	CheckRaw(name, value string)
}

type MakeVarsProvider func(ctx MakeVarsContext)

func RegisterMakeVarsProvider(pctx PackageContext, provider MakeVarsProvider) {
	makeVarsProviders = append(makeVarsProviders, makeVarsProvider{pctx, provider})
}

func init() {
	RegisterSingletonType("makevars", makeVarsSingletonFunc)
}

func makeVarsSingletonFunc() Singleton {
	return &makeVarsSingleton{}
}

type makeVarsSingleton struct{}

type makeVarsProvider struct {
	pctx PackageContext
	call MakeVarsProvider
}

var makeVarsProviders []makeVarsProvider

type makeVarsContext struct {
	config Config
	ctx    SingletonContext
	pctx   PackageContext
	vars   []makeVarsVariable
}

var _ MakeVarsContext = &makeVarsContext{}

type makeVarsVariable struct {
	name   string
	value  string
	sort   bool
	strict bool
}

func (s *makeVarsSingleton) GenerateBuildActions(ctx SingletonContext) {
	if !ctx.Config().EmbeddedInMake() {
		return
	}

	outFile := PathForOutput(ctx, "make_vars"+proptools.String(ctx.Config().productVariables.Make_suffix)+".mk").String()

	if ctx.Failed() {
		return
	}

	vars := []makeVarsVariable{}
	for _, provider := range makeVarsProviders {
		mctx := &makeVarsContext{
			config: ctx.Config(),
			ctx:    ctx,
			pctx:   provider.pctx,
		}

		provider.call(mctx)

		vars = append(vars, mctx.vars...)
	}

	if ctx.Failed() {
		return
	}

	outBytes := s.writeVars(vars)

	if _, err := os.Stat(outFile); err == nil {
		if data, err := ioutil.ReadFile(outFile); err == nil {
			if bytes.Equal(data, outBytes) {
				return
			}
		}
	}

	if err := ioutil.WriteFile(outFile, outBytes, 0666); err != nil {
		ctx.Errorf(err.Error())
	}
}

func (s *makeVarsSingleton) writeVars(vars []makeVarsVariable) []byte {
	buf := &bytes.Buffer{}

	fmt.Fprintln(buf, `# Autogenerated file

# Compares SOONG_$(1) against $(1), and warns if they are not equal.
#
# If the original variable is empty, then just set it to the SOONG_ version.
#
# $(1): Name of the variable to check
# $(2): If not-empty, sort the values before comparing
# $(3): Extra snippet to run if it does not match
define soong-compare-var
ifneq ($$($(1)),)
  my_val_make := $(if $(2),$$(sort $$(SOONG_$(1))),$$(SOONG_$(1)))
  my_val_soong := $(if $(2),$$(sort $$(SOONG_$(1))),$$(SOONG_$(1)))
  ifneq ($$(my_val_make),$$(my_val_soong))
    $$(warning $(1) does not match between Make and Soong:)
    $(if $(2),$$(warning Make  adds: $$(filter-out $$(my_val_soong),$$(my_val_make))),$$(warning Make : $$(my_val_make)))
    $(if $(2),$$(warning Soong adds: $$(filter-out $$(my_val_make),$$(my_val_soong))),$$(warning Soong: $$(my_val_soong)))
    $(3)
  endif
  my_val_make :=
  my_val_soong :=
else
  $(1) := $$(SOONG_$(1))
endif
# .KATI_READONLY := $(1) SOONG_$(1)
endef

my_check_failed := false`)

	for _, v := range vars {
		if !v.strict {
			continue
		}

		sort := ""
		if v.sort {
			sort = "true"
		}

		fmt.Fprintf(buf, "SOONG_%s := %s\n", v.name, v.value)
		fmt.Fprintf(buf, "$(eval $(call soong-compare-var,%s,%s,my_check_failed := true))\n\n", v.name, sort)
	}

	fmt.Fprintln(buf, `
ifneq ($(my_check_failed),false)
  $(error Soong variable check failed)
endif
my_check_failed :=`)

	for _, v := range vars {
		if v.strict {
			continue
		}

		sort := ""
		if v.sort {
			sort = "true"
		}

		fmt.Fprintf(buf, "SOONG_%s := %s\n", v.name, v.value)
		fmt.Fprintf(buf, "$(eval $(call soong-compare-var,%s,%s))\n\n", v.name, sort)
	}

	fmt.Fprintln(buf, "\nsoong-compare-var :=")

	return buf.Bytes()
}

func (c *makeVarsContext) Config() Config {
	return c.config
}

func (c *makeVarsContext) DeviceConfig() DeviceConfig {
	return DeviceConfig{c.config.deviceConfig}
}

func (c *makeVarsContext) SingletonContext() SingletonContext {
	return c.ctx
}

func (c *makeVarsContext) Eval(ninjaStr string) (string, error) {
	return c.ctx.Eval(c.pctx, ninjaStr)
}

func (c *makeVarsContext) addVariableRaw(name, value string, strict, sort bool) {
	c.vars = append(c.vars, makeVarsVariable{
		name:   name,
		value:  value,
		strict: strict,
		sort:   sort,
	})
}

func (c *makeVarsContext) addVariable(name, ninjaStr string, strict, sort bool) {
	value, err := c.Eval(ninjaStr)
	if err != nil {
		c.ctx.Errorf(err.Error())
	}
	c.addVariableRaw(name, value, strict, sort)
}

func (c *makeVarsContext) Strict(name, ninjaStr string) {
	c.addVariable(name, ninjaStr, false, false)
}
func (c *makeVarsContext) StrictSorted(name, ninjaStr string) {
	c.addVariable(name, ninjaStr, false, true)
}
func (c *makeVarsContext) StrictRaw(name, value string) {
	c.addVariableRaw(name, value, false, false)
}

func (c *makeVarsContext) Check(name, ninjaStr string) {
	c.addVariable(name, ninjaStr, false, false)
}
func (c *makeVarsContext) CheckSorted(name, ninjaStr string) {
	c.addVariable(name, ninjaStr, false, true)
}
func (c *makeVarsContext) CheckRaw(name, value string) {
	c.addVariableRaw(name, value, false, false)
}
