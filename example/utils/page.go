package utils

func GetTotalPage(total int64, page_size int64) int64 {
	// 计算总页数
	total_page := total / page_size
	if total%page_size > 0 {
		total_page++
	}
	return total_page
}
