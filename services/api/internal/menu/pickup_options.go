package menu

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type pickupOptionsResponse struct {
	Dates []pickupOptionsDateResponse `json:"dates"`
}
type pickupOptionsDateResponse struct {
	Date        string                      `json:"date"`
	Available   bool                        `json:"available"`
	MealPeriods []pickupOptionsMealResponse `json:"meal_periods"`
}
type pickupOptionsMealResponse struct {
	MealPeriod  MealCode `json:"meal_period"`
	Available   bool     `json:"available"`
	CutoffTime  string   `json:"cutoff_time"`
	PickupTimes []string `json:"pickup_times"`
}

func (handler *Handler) getPickupOptions(ctx *gin.Context) {
	now := handler.now().In(shanghaiLocation)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLocation)
	dates := []string{today.Format("2006-01-02"), today.AddDate(0, 0, 1).Format("2006-01-02")}
	facts, err := handler.reader.ReadPickupFacts(ctx.Request.Context(), dates)
	if err != nil || !validBusinessStatus(facts.BusinessStatus) {
		writeMenuUnavailable(ctx)
		return
	}
	response, err := newPickupOptionsResponse(facts, now)
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, response)
}

func newPickupOptionsResponse(facts PickupFacts, now time.Time) (pickupOptionsResponse, error) {
	periods, err := validateMealPeriods(facts.MealPeriods)
	if err != nil || facts.ServiceDates == nil {
		return pickupOptionsResponse{}, ErrMenuUnavailable
	}
	sort.SliceStable(periods, func(left, right int) bool { return periods[left].pickupStartMinute < periods[right].pickupStartMinute })
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLocation)
	result := pickupOptionsResponse{Dates: make([]pickupOptionsDateResponse, 0, 2)}
	for offset := 0; offset < 2; offset++ {
		date := today.AddDate(0, 0, offset)
		dateValue := date.Format("2006-01-02")
		serviceOpen, servicePresent := facts.ServiceDates[dateValue]
		dateResponse := pickupOptionsDateResponse{Date: dateValue, MealPeriods: make([]pickupOptionsMealResponse, 0, 2)}
		for _, period := range periods {
			cutoff := time.Date(date.Year(), date.Month(), date.Day(), period.cutoffMinute/60, period.cutoffMinute%60, 0, 0, shanghaiLocation)
			available := facts.BusinessStatus == "open" && servicePresent && serviceOpen && now.Before(cutoff)
			pickupTimes := make([]string, 0)
			for minute := period.pickupStartMinute; minute <= period.pickupEndMinute; minute += period.intervalMinutes {
				pickupTimes = append(pickupTimes, fmt.Sprintf("%02d:%02d", minute/60, minute%60))
			}
			dateResponse.Available = dateResponse.Available || available
			dateResponse.MealPeriods = append(dateResponse.MealPeriods, pickupOptionsMealResponse{
				MealPeriod: period.code, Available: available,
				CutoffTime: fmt.Sprintf("%02d:%02d", period.cutoffMinute/60, period.cutoffMinute%60), PickupTimes: pickupTimes,
			})
		}
		result.Dates = append(result.Dates, dateResponse)
	}
	return result, nil
}
