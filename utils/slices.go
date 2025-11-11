package utils

// Zip 遍历两个切片并执行函数
//
// 参数:
//   - a []T: 第一个切片
//   - b []U: 第二个切片
//   - fn func(T, U): 要执行的函数
//
// 说明:
//   - 两个切片长度不同时，以较短的为准
func Zip[T, U any](a []T, b []U, fn func(T, U)) {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		fn(a[i], b[i])
	}
}
