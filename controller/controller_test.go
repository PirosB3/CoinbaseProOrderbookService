package controller

import "testing"

func TestDateParsingWorks(t *testing.T) {
	dateString := "2020-10-11T20:50:02.941691Z"
	expectedResult := int64(1602449402)
	result, _ := DateStringToUnixEpoch(dateString)
	if result != expectedResult {
		t.Errorf("Expected %d but got %d", expectedResult, result)
	}
}
