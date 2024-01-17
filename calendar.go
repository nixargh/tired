package main

func calendar(year int, month int) int {
	// This is a map of working hours per month
	calendar := map[int]map[int]int{
		2024: {
			1:  136,
			2:  159,
			3:  159,
			4:  168,
			5:  159,
			6:  151,
			7:  184,
			8:  176,
			9:  168,
			10: 184,
			11: 167,
			12: 168,
		},
	}

	return calendar[year][month]
}
