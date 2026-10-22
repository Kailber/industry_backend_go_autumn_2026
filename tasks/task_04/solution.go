package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	firstDelta := nums[1] - nums[0]
	stats := Stats{
		Count: len(nums) - 1,
		Sum:   firstDelta,
		Min:   firstDelta,
		Max:   firstDelta,
	}

	previous := nums[1]
	for _, current := range nums[2:] {
		delta := current - previous
		stats.Sum += delta
		stats.Min = min(stats.Min, delta)
		stats.Max = max(stats.Max, delta)
		previous = current
	}

	return stats
}
