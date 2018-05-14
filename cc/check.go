package cc

import (
	"path/filepath"
	"strings"

	"android/soong/cc/config"
)

func CheckBadCompilerFlags(ctx BaseModuleContext, prop string, flags []string) {
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)

		if !strings.HasPrefix(flag, "-") {
			ctx.PropertyErrorf(prop, "Flag `%s` must start with `-`", flag)
		} else if strings.HasPrefix(flag, "-I") || strings.HasPrefix(flag, "-isystem") {
			ctx.PropertyErrorf(prop, "Bad flag `%s`, use local_include_dirs or include_dirs instead", flag)
		} else if inList(flag, config.IllegalFlags) {
			ctx.PropertyErrorf(prop, "Illegal flag `%s`", flag)
		} else if flag == "--coverage" {
			ctx.PropertyErrorf(prop, "Bad flag: `%s`, use native_coverage instead", flag)
		} else if strings.Contains(flag, " ") {
			args := strings.Split(flag, " ")
			if args[0] == "-include" {
				if len(args) > 2 {
					ctx.PropertyErrorf(prop, "`-include` only takes one argument: `%s`", flag)
				}
				path := filepath.Clean(args[1])
				if strings.HasPrefix("../", path) {
					ctx.PropertyErrorf(prop, "Path must not start with `../`: `%s`. Use include_dirs to -include from a different directory", flag)
				}
			} else if strings.HasPrefix(flag, "-D") && strings.Contains(flag, "=") {

			} else {
				ctx.PropertyErrorf(prop, "Bad flag: `%s` is not an allowed multi-word flag. Should it be split into multiple flags?", flag)
			}
		}
	}
}

func CheckBadLinkerFlags(ctx BaseModuleContext, prop string, flags []string) {
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)

		if !strings.HasPrefix(flag, "-") {
			ctx.PropertyErrorf(prop, "Flag `%s` must start with `-`", flag)
		} else if strings.HasPrefix(flag, "-l") {
			ctx.PropertyErrorf(prop, "Bad flag: `%s`, use shared_libs or host_ldlibs instead", flag)
		} else if strings.HasPrefix(flag, "-L") {
			ctx.PropertyErrorf(prop, "Bad flag: `%s` is not allowed", flag)
		} else if strings.HasPrefix(flag, "-Wl,--version-script") {
			ctx.PropertyErrorf(prop, "Bad flag: `%s`, use version_script instead", flag)
		} else if flag == "--coverage" {
			ctx.PropertyErrorf(prop, "Bad flag: `%s`, use native_coverage instead", flag)
		} else if strings.Contains(flag, " ") {
			args := strings.Split(flag, " ")
			if args[0] == "-z" {
				if len(args) > 2 {
					ctx.PropertyErrorf(prop, "`-z` only takes one argument: `%s`", flag)
				}
			} else {
				ctx.PropertyErrorf(prop, "Bad flag: `%s` is not an allowed multi-word flag. Should it be split into multiple flags?", flag)
			}
		}
	}
}

func CheckBadHostLdlibs(ctx ModuleContext, prop string, flags []string) {
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)

		if !strings.HasPrefix(flag, "-l") && !strings.HasPrefix(flag, "-framework") {
			ctx.PropertyErrorf(prop, "Invalid flag: `%s`, must start with `-l` or `-framework`", flag)
		}
	}
}

func CheckBadTidyFlags(ctx ModuleContext, prop string, flags []string) {
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)

		if !strings.HasPrefix(flag, "-") {
			ctx.PropertyErrorf(prop, "Flag `%s` must start with `-`", flag)
		} else if strings.HasPrefix(flag, "-fix") {
			ctx.PropertyErrorf(prop, "Flag `%s` is not allowed, since it could cause multiple writes to the same source file", flag)
		} else if strings.HasPrefix(flag, "-checks=") {
			ctx.PropertyErrorf(prop, "Flag `%s` is not allowed, use `tidy_checks` property instead", flag)
		} else if strings.Contains(flag, " ") {
			ctx.PropertyErrorf(prop, "Bad flag: `%s` is not an allowed multi-word flag. Should it be split into multiple flags?", flag)
		}
	}
}

func CheckBadTidyChecks(ctx ModuleContext, prop string, checks []string) {
	for _, check := range checks {
		if strings.Contains(check, " ") {
			ctx.PropertyErrorf("tidy_checks", "Check `%s` invalid, cannot contain spaces", check)
		} else if strings.Contains(check, ",") {
			ctx.PropertyErrorf("tidy_checks", "Check `%s` invalid, cannot contain commas. Split each entry into it's own string instead", check)
		}
	}
}
