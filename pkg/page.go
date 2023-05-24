package pkg

func Paginate[T any](items []T, page int, pageSize int) []T {
	start := page * pageSize
	end := (page + 1) * pageSize

	if start >= len(items) {
		return []T{}
	}

	if end > len(items) {
		end = len(items)
	}

	return items[start:end]
}
