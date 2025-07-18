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
		2025: {
			1:  136,
			2:  160,
			3:  167,
			4:  175,
			5:  144,
			6:  151,
			7:  184,
			8:  168,
			9:  176,
			10: 184,
			11: 151,
			12: 176,
		},
	}

	return calendar[year][month]
}
