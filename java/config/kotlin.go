package config

var (
	KotlinStdlibJar = "external/kotlinc/lib/kotlin-stdlib.jar"
)

func init() {
	pctx.StaticVariable("KotlincCmd", "external/kotlinc/bin/kotlinc")
	pctx.StaticVariable("KotlinCompilerJar", "external/kotlinc/lib/kotlin-compiler.jar")
	pctx.StaticVariable("KotlinStdlibJar", KotlinStdlibJar)
}
