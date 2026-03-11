package varx

import "slices"

func ArrContains[T comparable](arr []T, val T) bool {
	return slices.Contains(arr, val)
}
