package android

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func JoinWithPrefix(strs []string, prefix string) string {
	if len(strs) == 0 {
		return ""
	}

	if len(strs) == 1 {
		return prefix + strs[0]
	}

	n := len(" ") * (len(strs) - 1)
	for _, s := range strs {
		n += len(prefix) + len(s)
	}

	ret := make([]byte, 0, n)
	for i, s := range strs {
		if i != 0 {
			ret = append(ret, ' ')
		}
		ret = append(ret, prefix...)
		ret = append(ret, s...)
	}
	return string(ret)
}

func sortedKeys(m map[string][]string) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

func IndexList(s string, list []string) int {
	for i, l := range list {
		if l == s {
			return i
		}
	}
	return -1
}

func InList(s string, list []string) bool {
	return IndexList(s, list) != -1
}

func PrefixInList(s string, list []string) bool {
	for _, prefix := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func FilterList(list []string, filter []string) (remainder []string, filtered []string) {
	for _, l := range list {
		if InList(l, filter) {
			filtered = append(filtered, l)
		} else {
			remainder = append(remainder, l)
		}
	}
	return
}

func RemoveListFromList(list []string, filter_out []string) (result []string) {
	result = make([]string, 0, len(list))
	for _, l := range list {
		if !InList(l, filter_out) {
			result = append(result, l)
		}
	}
	return
}

func RemoveFromList(s string, list []string) (bool, []string) {
	i := IndexList(s, list)
	if i == -1 {
		return false, list
	}

	result := make([]string, 0, len(list)-1)
	result = append(result, list[:i]...)
	for _, l := range list[i+1:] {
		if l != s {
			result = append(result, l)
		}
	}
	return true, result
}

func FirstUniqueStrings(list []string) []string {
	k := 0
outer:
	for i := 0; i < len(list); i++ {
		for j := 0; j < k; j++ {
			if list[i] == list[j] {
				continue outer
			}
		}
		list[k] = list[i]
		k++
	}
	return list[:k]
}

func LastUniqueStrings(list []string) []string {
	totalSkip := 0
	for i := len(list) - 1; i >= totalSkip; i-- {
		skip := 0
		for j := i - 1; j >= totalSkip; j-- {
			if list[i] == list[j] {
				skip++
			} else {
				list[j+skip] = list[j]
			}
		}
		totalSkip += skip
	}
	return list[totalSkip:]
}

func checkCalledFromInit() {
	for skip := 3; ; skip++ {
		_, funcName, ok := callerName(skip)
		if !ok {
			panic("not called from an init func")
		}

		if funcName == "init" || strings.HasPrefix(funcName, "init·") {
			return
		}
	}
}

func callerName(skip int) (pkgPath, funcName string, ok bool) {
	var pc [1]uintptr
	n := runtime.Callers(skip+1, pc[:])
	if n != 1 {
		return "", "", false
	}

	f := runtime.FuncForPC(pc[0])
	fullName := f.Name()

	lastDotIndex := strings.LastIndex(fullName, ".")
	if lastDotIndex == -1 {
		panic("unable to distinguish function name from package")
	}

	if fullName[lastDotIndex-1] == ')' {
		lastDotIndex = strings.LastIndex(fullName[:lastDotIndex], ".")
	}

	pkgPath = fullName[:lastDotIndex]
	funcName = fullName[lastDotIndex+1:]
	ok = true
	return
}

func GetNumericSdkVersion(v string) string {
	if strings.Contains(v, "system_") {
		return strings.Replace(v, "system_", "", 1)
	}
	return v
}

func Prefix() string {
	return "/data/data/com.termux/files/usr"
}

func TermuxExecutable(exec string) string {
	return Prefix() + "/bin/" + exec
}

func StringToPath(pathComponents ...string) Path {
	path := filepath.Join(pathComponents...)
	ret := basePath{path: path, rel: "/"}
	return ret
}
