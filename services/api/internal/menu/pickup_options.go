package menu

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type pickupOptionsResponse struct {
	Timezone string                      `json:"timezone"`
	Dates    []pickupOptionsDateResponse `json:"dates"`
}

type pickupOptionsDateResponse struct {
	Date      string                      `json:"date"`
	Orderable bool                        `json:"orderable"`
	Meals     []pickupOptionsMealResponse `json:"meals"`
}

type pickupOptionsMealResponse struct {
	Code        MealCode `json:"code"`
	CutoffAt    string   `json:"cutoff_at"`
	Orderable   bool     `json:"orderable"`
	PickupTimes []string `json:"pickup_times"`
}

func (handler *Handler) getPickupOptions(ctx *gin.Context) {
	now := handler.now().In(shanghaiLocation)
	records, err := handler.reader.MealPeriods(ctx.Request.Context())
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}
	response, err := newPickupOptionsResponse(records, now)
	if err != nil {
		writeMenuUnavailable(ctx)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, response)
}

func newPickupOptionsResponse(records []MealPeriodRecord, now time.Time) (pickupOptionsResponse, error) {
	periods, err := validateMealPeriods(records)
	if err != nil {
		return pickupOptionsResponse{}, err
	}
	sort.SliceStable(periods, func(left, right int) bool {
		return periods[left].pickupStartMinute < periods[right].pickupStartMinute
	})
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLocation)
	dates := make([]pickupOptionsDateResponse, 0, 2)
	for dayOffset := 0; dayOffset < 2; dayOffset++ {
		date := today.AddDate(0, 0, dayOffset)
		meals := make([]pickupOptionsMealResponse, 0, len(periods))
		dateOrderable := false
		for _, period := range periods {
			cutoffAt := time.Date(
				date.Year(), date.Month(), date.Day(),
				period.cutoffMinute/60, period.cutoffMinute%60, 0, 0, shanghaiLocation,
			)
			pickupTimes := make([]string, 0)
			for minute := period.pickupStartMinute; minute <= period.pickupEndMinute; minute += period.intervalMinutes {
				pickupTimes = append(pickupTimes, fmt.Sprintf("%02d:%02d", minute/60, minute%60))
			}
			mealOrderable := now.Before(cutoffAt)
			dateOrderable = dateOrderable || mealOrderable
			meals = append(meals, pickupOptionsMealResponse{
				Code: period.code, CutoffAt: cutoffAt.Format(time.RFC3339), Orderable: mealOrderable, PickupTimes: pickupTimes,
			})
		}
		dates = append(dates, pickupOptionsDateResponse{
			Date: date.Format("2006-01-02"), Orderable: dateOrderable, Meals: meals,
		})
	}
	return pickupOptionsResponse{Timezone: menuTimezone, Dates: dates}, nil
}
