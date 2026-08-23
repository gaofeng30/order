package quote

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateRejectsClientOwnedMoneyWithoutCallingApplication(t *testing.T) {
	application := &quoteApplicationStub{}
	router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, application))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":2,"price_cents":1}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque-session")
	request.Header.Set("Idempotency-Key", "quote-attempt-1")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assertQuoteResponse(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`)
	if application.createCalls != 0 {
		t.Fatalf("application create calls = %d", application.createCalls)
	}
}

func TestCreateRejectsDuplicateJSONKeysAtEveryObjectDepth(t *testing.T) {
	tests := []string{
		`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_date":"2026-08-25","pickup_time":"12:00","items":[{"product_id":"8","quantity":2}]}`,
		`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","product_id":"9","quantity":2}]}`,
	}
	for _, body := range tests {
		application := &quoteApplicationStub{}
		router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, application))
		request := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer opaque-session")
		request.Header.Set("Idempotency-Key", "quote-attempt-1")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertQuoteResponse(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`)
		if application.createCalls != 0 {
			t.Fatalf("application create calls = %d", application.createCalls)
		}
	}
}

func TestCreateMapsStablePIIFreeApplicationErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "invalid", err: ErrInvalidInput, status: http.StatusBadRequest, body: `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`},
		{name: "primary phone", err: ErrPrimaryPhoneRequired, status: http.StatusConflict, body: `{"error":{"code":"PRIMARY_PHONE_REQUIRED","message":"primary phone required"}}`},
		{name: "selection", err: ErrSelectionUnavailable, status: http.StatusConflict, body: `{"error":{"code":"QUOTE_SELECTION_UNAVAILABLE","message":"quote selection unavailable"}}`},
		{name: "idempotency", err: ErrIdempotencyConflict, status: http.StatusConflict, body: `{"error":{"code":"IDEMPOTENCY_KEY_CONFLICT","message":"idempotency key conflict"}}`},
		{name: "payment amount", err: ErrPaymentAmountTooSmall, status: http.StatusConflict, body: `{"error":{"code":"PAYMENT_AMOUNT_TOO_SMALL","message":"payment amount too small"}}`},
		{name: "unknown", err: errors.New("sql: phone +1234567890"), status: http.StatusServiceUnavailable, body: `{"error":{"code":"QUOTE_UNAVAILABLE","message":"quote temporarily unavailable"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			application := &quoteApplicationStub{createErr: test.err}
			router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, application))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":2}]}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer opaque-session")
			request.Header.Set("Idempotency-Key", "quote-attempt-1")
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assertQuoteResponse(t, response, test.status, test.body)
			if strings.Contains(response.Body.String(), "+1234567890") || strings.Contains(response.Body.String(), "sql") {
				t.Fatalf("response disclosed sensitive error = %q", response.Body.String())
			}
		})
	}
}

func TestReadHidesAbsentAndNonOwnedQuotesAsNotFound(t *testing.T) {
	for _, applicationErr := range []error{ErrNotFound, ErrForbidden} {
		application := &quoteApplicationStub{readErr: applicationErr}
		router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, application))
		request := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/91", nil)
		request.Header.Set("Authorization", "Bearer opaque-session")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertQuoteResponse(t, response, http.StatusNotFound, `{"error":{"code":"QUOTE_NOT_FOUND","message":"quote not found"}}`)
	}
}

func TestReadRejectsRequestBody(t *testing.T) {
	application := &quoteApplicationStub{}
	router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, application))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/91", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer opaque-session")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	assertQuoteResponse(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_REQUEST","message":"invalid request"}}`)
	if application.readCalls != 0 {
		t.Fatalf("application read calls = %d", application.readCalls)
	}
}

func TestHandlerFailsClosedOnInvalidApplicationSnapshot(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, &quoteApplicationStub{
			createResult: CreateResult{Quote: Quote{ID: 91, UserID: 42}, Created: true},
		}))
		request := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":2}]}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer opaque-session")
		request.Header.Set("Idempotency-Key", "quote-attempt-1")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertQuoteResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"QUOTE_UNAVAILABLE","message":"quote temporarily unavailable"}}`)
	})

	t.Run("read", func(t *testing.T) {
		router := quoteTestRouter(NewHandler(&sessionAuthenticatorStub{userID: 42}, &quoteApplicationStub{
			readQuote: Quote{ID: 91, UserID: 43},
		}))
		request := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/91", nil)
		request.Header.Set("Authorization", "Bearer opaque-session")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		assertQuoteResponse(t, response, http.StatusServiceUnavailable, `{"error":{"code":"QUOTE_UNAVAILABLE","message":"quote temporarily unavailable"}}`)
	})
}
