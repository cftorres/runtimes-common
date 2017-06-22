package differs

import "errors"

var diffs = map[string]func(string, string) string{
	"hist": History,
	"dir":  Package,
}

func Diff(arg1, arg2, differ string) (string, error) {
	if f, exists := diffs[differ]; exists {
		return callDiffer(arg1, arg2, f), nil
	} else {
		return "", errors.New("Unknown differ.")
	}
}

func callDiffer(arg1, arg2 string, differ func(string, string) string) string {
	return differ(arg1, arg2)
}
