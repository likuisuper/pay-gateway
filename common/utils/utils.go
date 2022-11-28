package utils

import "fmt"

func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch result := v.(type) {
	case string:
		return result
	case []byte:
		return string(result)
	default:
		return fmt.Sprint(result)
	}
}
