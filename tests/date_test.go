package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chiGen "github.com/kolah/eugene/v2/tests/generated/e2e_chi"
	basic "github.com/kolah/eugene/v2/tests/generated/e2e_echo"
	stdlibGen "github.com/kolah/eugene/v2/tests/generated/e2e_stdlib"
	strict "github.com/kolah/eugene/v2/tests/generated/e2e_strict_echo"
	formats "github.com/kolah/eugene/v2/tests/generated/types_formats"
)

func TestDateFieldRoundTrip(t *testing.T) {
	const payload = `{"updatedAt":"2024-01-02"}`

	var record formats.Record
	require.NoError(t, json.Unmarshal([]byte(payload), &record))

	got, err := json.Marshal(record)
	require.NoError(t, err)
	assert.Equal(t, payload, string(got))
}

func TestDateUnionVariantRoundTrip(t *testing.T) {
	const payload = `"2024-01-02"`

	var union basic.DayOrCount
	require.NoError(t, json.Unmarshal([]byte(payload), &union))

	day, err := union.AsDay()
	require.NoError(t, err)
	assert.Equal(t, basic.Day("2024-01-02"), *day)

	got, err := json.Marshal(union)
	require.NoError(t, err)
	assert.Equal(t, payload, string(got))
}

func TestDateParamsRoundTrip(t *testing.T) {
	e := echo.New()
	basic.RegisterHandlers(e, &BasicEchoHandler{})

	strictEcho := echo.New()
	strict.RegisterStrictHandlers(strictEcho, &StrictEchoHandler{})

	servers := map[string]http.Handler{
		"echo":   e,
		"strict": strictEcho,
		"chi":    chiGen.Handler(&ChiHandler{}),
		"stdlib": stdlibGen.Handler(&StdlibHandler{}),
	}

	for name, handler := range servers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/days/2024-01-02?since=2024-01-03", nil)
			req.Header.Set("X-Until", "2024-01-04")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.JSONEq(t, `{"day":"2024-01-02","since":"2024-01-03","until":"2024-01-04"}`, rec.Body.String())
		})
	}
}

func TestDateClientParams(t *testing.T) {
	e := echo.New()
	basic.RegisterHandlers(e, &BasicEchoHandler{})
	server := httptest.NewServer(e)
	defer server.Close()

	since, until := "2024-01-03", "2024-01-04"
	resp, err := basic.NewClient(server.URL).GetDay(context.Background(), "2024-01-02", &basic.GetDayParams{
		Since:  &since,
		XUntil: &until,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.JSON200)
	assert.Equal(t, basic.DayRecord{Day: "2024-01-02", Since: &since, Until: &until}, *resp.JSON200)
}

func (h *BasicEchoHandler) GetDay(ctx echo.Context, day string, params basic.GetDayQueryParams, headers basic.GetDayHeaderParams) error {
	return ctx.JSON(http.StatusOK, basic.DayRecord{Day: day, Since: params.Since, Until: headers.XUntil})
}

func (h *StrictEchoHandler) GetDay(ctx context.Context, req strict.GetDayRequestObject) (strict.GetDayResponseObject, error) {
	return strict.GetDay200JSONResponse{Day: req.Day, Since: req.Since, Until: req.XUntil}, nil
}

func (h *ChiHandler) GetDay(w http.ResponseWriter, r *http.Request, day string, params chiGen.GetDayQueryParams, headers chiGen.GetDayHeaderParams) {
	json.NewEncoder(w).Encode(chiGen.DayRecord{Day: day, Since: params.Since, Until: headers.XUntil})
}

func (h *StdlibHandler) GetDay(w http.ResponseWriter, r *http.Request, day string, params stdlibGen.GetDayQueryParams, headers stdlibGen.GetDayHeaderParams) {
	json.NewEncoder(w).Encode(stdlibGen.DayRecord{Day: day, Since: params.Since, Until: headers.XUntil})
}
