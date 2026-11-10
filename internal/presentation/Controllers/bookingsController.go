package presentation

import (
	"encoding/json"
	"net/http"
)

type BookingController struct {
}

type GetBookingAvailabilityRequest struct {
	PCamount int
	Zone     string
	Together bool
}

func (bc *BookingController) GetBookingAvailability(w http.ResponseWriter, r *http.Request) {
	var req GetBookingAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}

}
