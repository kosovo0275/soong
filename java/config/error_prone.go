package config

import "android/soong/android"

var (
	// These will be filled out by external/error_prone/soong/error_prone.go if it is available
	ErrorProneJavacJar    string
	ErrorProneJar         string
	ErrorProneClasspath   string
	ErrorProneChecksError string
	ErrorProneFlags       string
)

// Wrapper that grabs value of val late so it can be initialized by a later module's init function
func errorProneVar(name string, val *string) {
	pctx.VariableFunc(name, func(android.PackageVarContext) string {
		return *val
	})
}

func init() {
	errorProneVar("ErrorProneJar", &ErrorProneJar)
	errorProneVar("ErrorProneJavacJar", &ErrorProneJavacJar)
	errorProneVar("ErrorProneClasspath", &ErrorProneClasspath)
	errorProneVar("ErrorProneChecksError", &ErrorProneChecksError)
	errorProneVar("ErrorProneFlags", &ErrorProneFlags)

	pctx.StaticVariable("ErrorProneCmd",
		"${JavaCmd} -Xmx${JavacHeapSize} -Xbootclasspath/p:${ErrorProneJavacJar} "+
			"-cp ${ErrorProneJar}:${ErrorProneClasspath} "+
			"${ErrorProneFlags} ${CommonJdkFlags} ${ErrorProneChecksError}")

}
