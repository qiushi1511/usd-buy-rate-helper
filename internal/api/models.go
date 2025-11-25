package api

import (
	"errors"
	"fmt"
	"strconv"
)

// CMBResponse represents the top-level response from CMB API
type CMBResponse struct {
	ReturnCode string   `json:"returnCode"`
	ErrorMsg   *string  `json:"errorMsg"`
	Body       *CMBBody `json:"body"`
}

// CMBBody contains the actual exchange rate data
type CMBBody struct {
	Data []CMBCurrencyRate `json:"data"`
	Time string            `json:"time"`
}

// CMBCurrencyRate represents a single currency's exchange rate
type CMBCurrencyRate struct {
	CcyNbr string `json:"ccyNbr"` // Currency name in Chinese (e.g., "美元")
	RtbBid string `json:"rtbBid"` // Bank bid
	RthOfr string `json:"rthOfr"` // Cash offer
	RtcOfr string `json:"rtcOfr"` // Cash offer
	RthBid string `json:"rthBid"` // Cash bid
	RtcBid string `json:"rtcBid"` // Cash bid rate - THIS IS WHAT WE NEED
	RatTim string `json:"ratTim"` // Rate time
	RatDat string `json:"ratDat"` // Rate date
	CcyExc string `json:"ccyExc"` // Exchange unit (e.g., "10")
}

// ExtractUSDRate extracts the USD exchange rate from CMB API response
// Returns the rate divided by 100 (since ccyExc is "10")
func ExtractUSDRate(resp *CMBResponse) (float64, error) {
	if resp.ReturnCode != "SUC0000" {
		errMsg := "unknown error"
		if resp.ErrorMsg != nil {
			errMsg = *resp.ErrorMsg
		}
		return 0, fmt.Errorf("API error: %s", errMsg)
	}

	if resp.Body == nil || len(resp.Body.Data) == 0 {
		return 0, errors.New("empty response body")
	}

	for _, rate := range resp.Body.Data {
		if rate.CcyNbr == "美元" {
			// Parse string to float
			val, err := strconv.ParseFloat(rate.RtcBid, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing rate: %w", err)
			}

			// Divide by 100 (since ccyExc is "10", rates are per 10 units)
			return val / 100, nil
		}
	}

	return 0, errors.New("USD (美元) not found in response")
}
