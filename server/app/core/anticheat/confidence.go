package anticheat

import "osdtyp/app/utils"

const standard_deviation_confidence int = 2
const shortest_interval_confidence int = 4

// Todo: Test normal deviations in human typing vs automated ones
func (a *AntiCheat) StandardDeviationTest(time_diff []int32) int {
	sd := utils.StandardDeviation(time_diff)
	if sd < 15 {
		return -standard_deviation_confidence
	}
	return standard_deviation_confidence
}
func (a *AntiCheat) ShortestInterval(time_diff []int32) int {
	shortest_span := utils.FindMinIgnoringFirst(time_diff)
	if shortest_span < 40 { //Under 40 milliseconds is probably automated
		return -shortest_interval_confidence
	}
	return shortest_interval_confidence
}
