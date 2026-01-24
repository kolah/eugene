package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	basic "github.com/kolah/eugene/tests/generated/e2e_echo"
	chiGen "github.com/kolah/eugene/tests/generated/e2e_chi"
	stdlibGen "github.com/kolah/eugene/tests/generated/e2e_stdlib"
	strict "github.com/kolah/eugene/tests/generated/e2e_strict_echo"
)

// === Basic Server Handler ===

type BasicEchoHandler struct{}

func (h *BasicEchoHandler) EchoJSON(ctx echo.Context) error {
	var body basic.EchoPayload
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return ctx.JSON(http.StatusOK, body)
}

func (h *BasicEchoHandler) EchoForm(ctx echo.Context, req basic.EchoFormFormRequest) error {
	field2, _ := strconv.Atoi(req.Field2)
	return ctx.JSON(http.StatusOK, basic.FormEchoResponse{
		ReceivedField1: &req.Field1,
		ReceivedField2: &field2,
		ReceivedTags:   req.Tags,
	})
}

func (h *BasicEchoHandler) EchoMultipart(ctx echo.Context, req basic.EchoMultipartMultipartRequest) error {
	var filename string
	var size int
	if req.File != nil {
		filename = req.File.Filename
		size = int(req.File.Size)
	}
	return ctx.JSON(http.StatusOK, basic.FileEchoResponse{
		Filename:    &filename,
		Size:        &size,
		Description: &req.Description,
	})
}

func (h *BasicEchoHandler) GetItem(ctx echo.Context, id string, params basic.GetItemQueryParams) error {
	requestID := ctx.Request().Header.Get("X-Request-ID")

	if id == "not-found" {
		code := "NOT_FOUND"
		msg := "Item not found"
		return ctx.JSON(http.StatusNotFound, basic.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
	}

	return ctx.JSON(http.StatusOK, basic.ItemWithParams{
		ID:        &id,
		Filter:    params.Filter,
		RequestID: &requestID,
	})
}

func (h *BasicEchoHandler) CreateResource(ctx echo.Context) error {
	var body basic.NewResource
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	id := "res-123"
	return ctx.JSON(http.StatusCreated, basic.Resource{
		ID:          &id,
		Name:        &body.Name,
		Status:      (*basic.Status)(body.Status),
		Description: body.Description,
	})
}

func (h *BasicEchoHandler) DeleteResource(ctx echo.Context, id string) error {
	return ctx.NoContent(http.StatusNoContent)
}

func (h *BasicEchoHandler) GetSession(ctx echo.Context) error {
	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		code := "MISSING_COOKIE"
		msg := "session_id cookie required"
		return ctx.JSON(http.StatusBadRequest, basic.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
	}
	userID := "user-456"
	expiresAt := "2025-12-31T23:59:59Z"
	return ctx.JSON(http.StatusOK, basic.SessionInfo{
		SessionID: &sessionID.Value,
		UserID:    &userID,
		ExpiresAt: &expiresAt,
	})
}

func (h *BasicEchoHandler) GetSecureData(ctx echo.Context) error {
	apiKey := ctx.Request().Header.Get("X-API-Key")
	if apiKey != "valid-api-key" {
		code := "UNAUTHORIZED"
		msg := "Invalid API key"
		return ctx.JSON(http.StatusUnauthorized, basic.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
	}
	secret := "top-secret-data"
	level := "admin"
	return ctx.JSON(http.StatusOK, basic.SecureData{
		Secret:      &secret,
		AccessLevel: &level,
	})
}

func (h *BasicEchoHandler) CreateShape(ctx echo.Context) error {
	var body basic.Shape
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return ctx.JSON(http.StatusOK, body)
}

// === Strict Server Handler ===

type StrictEchoHandler struct{}

func (h *StrictEchoHandler) EchoJSON(ctx context.Context, req strict.EchoJSONRequestObject) (strict.EchoJSONResponseObject, error) {
	return strict.EchoJSON200JSONResponse(req.Body), nil
}

func (h *StrictEchoHandler) EchoForm(ctx context.Context, req strict.EchoFormRequestObject) (strict.EchoFormResponseObject, error) {
	return strict.EchoForm200JSONResponse{}, nil
}

func (h *StrictEchoHandler) EchoMultipart(ctx context.Context, req strict.EchoMultipartRequestObject) (strict.EchoMultipartResponseObject, error) {
	return strict.EchoMultipart200JSONResponse{}, nil
}

func (h *StrictEchoHandler) GetItem(ctx context.Context, req strict.GetItemRequestObject) (strict.GetItemResponseObject, error) {
	if req.ID == "not-found" {
		code := "NOT_FOUND"
		msg := "Item not found"
		return strict.GetItem404JSONResponse{
			Code:    &code,
			Message: &msg,
		}, nil
	}

	return strict.GetItem200JSONResponse{
		ID:        &req.ID,
		Filter:    req.Filter,
		RequestID: req.XRequestID,
	}, nil
}

func (h *StrictEchoHandler) CreateResource(ctx context.Context, req strict.CreateResourceRequestObject) (strict.CreateResourceResponseObject, error) {
	id := "res-123"
	return strict.CreateResource201JSONResponse{
		ID:          &id,
		Name:        &req.Body.Name,
		Status:      (*strict.Status)(req.Body.Status),
		Description: req.Body.Description,
	}, nil
}

func (h *StrictEchoHandler) DeleteResource(ctx context.Context, req strict.DeleteResourceRequestObject) (strict.DeleteResourceResponseObject, error) {
	return strict.DeleteResource204Response{}, nil
}

func (h *StrictEchoHandler) GetSession(ctx context.Context) (strict.GetSessionResponseObject, error) {
	sessionID := "session-from-cookie"
	userID := "user-456"
	expiresAt := "2025-12-31T23:59:59Z"
	return strict.GetSession200JSONResponse{
		SessionID: &sessionID,
		UserID:    &userID,
		ExpiresAt: &expiresAt,
	}, nil
}

func (h *StrictEchoHandler) GetSecureData(ctx context.Context) (strict.GetSecureDataResponseObject, error) {
	secret := "top-secret-data"
	level := "admin"
	return strict.GetSecureData200JSONResponse{
		Secret:      &secret,
		AccessLevel: &level,
	}, nil
}

func (h *StrictEchoHandler) CreateShape(ctx context.Context, req strict.CreateShapeRequestObject) (strict.CreateShapeResponseObject, error) {
	return strict.CreateShape200JSONResponse(req.Body), nil
}

// === Chi Server Handler ===

type ChiHandler struct{}

func (h *ChiHandler) EchoJSON(w http.ResponseWriter, r *http.Request) {
	var body chiGen.EchoPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

func (h *ChiHandler) EchoForm(w http.ResponseWriter, r *http.Request, req chiGen.EchoFormFormRequest) {
	field2, _ := strconv.Atoi(req.Field2)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chiGen.FormEchoResponse{
		ReceivedField1: &req.Field1,
		ReceivedField2: &field2,
		ReceivedTags:   req.Tags,
	})
}

func (h *ChiHandler) EchoMultipart(w http.ResponseWriter, r *http.Request, req chiGen.EchoMultipartMultipartRequest) {
	var filename string
	var size int
	if req.File != nil {
		filename = req.File.Filename
		size = int(req.File.Size)
	}
	desc := r.FormValue("description")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chiGen.FileEchoResponse{
		Filename:    &filename,
		Size:        &size,
		Description: &desc,
	})
}

func (h *ChiHandler) GetItem(w http.ResponseWriter, r *http.Request, id string, params chiGen.GetItemQueryParams) {
	requestID := r.Header.Get("X-Request-ID")

	if id == "not-found" {
		code := "NOT_FOUND"
		msg := "Item not found"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(chiGen.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chiGen.ItemWithParams{
		ID:        &id,
		Filter:    params.Filter,
		RequestID: &requestID,
	})
}

func (h *ChiHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var body chiGen.NewResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := "res-123"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chiGen.Resource{
		ID:          &id,
		Name:        &body.Name,
		Status:      (*chiGen.Status)(body.Status),
		Description: body.Description,
	})
}

func (h *ChiHandler) DeleteResource(w http.ResponseWriter, r *http.Request, id string) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *ChiHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		code := "MISSING_COOKIE"
		msg := "session_id cookie required"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(chiGen.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
		return
	}
	userID := "user-456"
	expiresAt := "2025-12-31T23:59:59Z"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chiGen.SessionInfo{
		SessionID: &cookie.Value,
		UserID:    &userID,
		ExpiresAt: &expiresAt,
	})
}

func (h *ChiHandler) GetSecureData(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "valid-api-key" {
		code := "UNAUTHORIZED"
		msg := "Invalid API key"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(chiGen.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
		return
	}
	secret := "top-secret-data"
	level := "admin"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chiGen.SecureData{
		Secret:      &secret,
		AccessLevel: &level,
	})
}

func (h *ChiHandler) CreateShape(w http.ResponseWriter, r *http.Request) {
	var body chiGen.Shape
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

// === Stdlib Server Handler ===

type StdlibHandler struct{}

func (h *StdlibHandler) EchoJSON(w http.ResponseWriter, r *http.Request) {
	var body stdlibGen.EchoPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

func (h *StdlibHandler) EchoForm(w http.ResponseWriter, r *http.Request, req stdlibGen.EchoFormFormRequest) {
	field2, _ := strconv.Atoi(req.Field2)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stdlibGen.FormEchoResponse{
		ReceivedField1: &req.Field1,
		ReceivedField2: &field2,
		ReceivedTags:   req.Tags,
	})
}

func (h *StdlibHandler) EchoMultipart(w http.ResponseWriter, r *http.Request, req stdlibGen.EchoMultipartMultipartRequest) {
	var filename string
	var size int
	if req.File != nil {
		filename = req.File.Filename
		size = int(req.File.Size)
	}
	desc := r.FormValue("description")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stdlibGen.FileEchoResponse{
		Filename:    &filename,
		Size:        &size,
		Description: &desc,
	})
}

func (h *StdlibHandler) GetItem(w http.ResponseWriter, r *http.Request, id string, params stdlibGen.GetItemQueryParams) {
	requestID := r.Header.Get("X-Request-ID")

	if id == "not-found" {
		code := "NOT_FOUND"
		msg := "Item not found"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(stdlibGen.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stdlibGen.ItemWithParams{
		ID:        &id,
		Filter:    params.Filter,
		RequestID: &requestID,
	})
}

func (h *StdlibHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var body stdlibGen.NewResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := "res-123"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(stdlibGen.Resource{
		ID:          &id,
		Name:        &body.Name,
		Status:      (*stdlibGen.Status)(body.Status),
		Description: body.Description,
	})
}

func (h *StdlibHandler) DeleteResource(w http.ResponseWriter, r *http.Request, id string) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *StdlibHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		code := "MISSING_COOKIE"
		msg := "session_id cookie required"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(stdlibGen.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
		return
	}
	userID := "user-456"
	expiresAt := "2025-12-31T23:59:59Z"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stdlibGen.SessionInfo{
		SessionID: &cookie.Value,
		UserID:    &userID,
		ExpiresAt: &expiresAt,
	})
}

func (h *StdlibHandler) GetSecureData(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "valid-api-key" {
		code := "UNAUTHORIZED"
		msg := "Invalid API key"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(stdlibGen.ErrorResponse{
			Code:    &code,
			Message: &msg,
		})
		return
	}
	secret := "top-secret-data"
	level := "admin"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stdlibGen.SecureData{
		Secret:      &secret,
		AccessLevel: &level,
	})
}

func (h *StdlibHandler) CreateShape(w http.ResponseWriter, r *http.Request) {
	var body stdlibGen.Shape
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

// === Tests ===

func TestE2EBasicServer(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	client := basic.NewClient(server.URL)
	ctx := context.Background()

	t.Run("JSON round-trip", func(t *testing.T) {
		msg := "hello"
		num := 42
		nestedVal := "nested-value"
		resp, err := client.EchoJSON(ctx, basic.EchoPayload{
			Message: msg,
			Number:  &num,
			Nested: basic.EchoPayloadNested{
				Value: &nestedVal,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, msg, resp.JSON200.Message)
		assert.Equal(t, num, *resp.JSON200.Number)
		assert.Equal(t, nestedVal, *resp.JSON200.Nested.Value)
	})

	t.Run("Form round-trip", func(t *testing.T) {
		resp, err := client.EchoForm(ctx, basic.EchoFormRequest{
			Field1: "test-field",
			Field2: "123",
			Tags:   []string{"tag1", "tag2"},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "test-field", *resp.JSON200.ReceivedField1)
		assert.Equal(t, 123, *resp.JSON200.ReceivedField2)
		assert.Equal(t, []string{"tag1", "tag2"}, resp.JSON200.ReceivedTags)
	})

	t.Run("Multipart round-trip", func(t *testing.T) {
		fileContent := []byte("test file content")
		resp, err := client.EchoMultipart(ctx, basic.EchoMultipartRequest{
			File: &basic.FileUpload{
				Reader:   bytes.NewReader(fileContent),
				Filename: "test.txt",
			},
			Description: "test description",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "test.txt", *resp.JSON200.Filename)
		assert.Equal(t, len(fileContent), *resp.JSON200.Size)
		assert.Equal(t, "test description", *resp.JSON200.Description)
	})

	t.Run("Path and query params", func(t *testing.T) {
		filter := "active"
		resp, err := client.GetItem(ctx, "item-123", &basic.GetItemParams{
			Filter: &filter,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "item-123", *resp.JSON200.ID)
		assert.Equal(t, "active", *resp.JSON200.Filter)
	})

	t.Run("404 error response", func(t *testing.T) {
		resp, err := client.GetItem(ctx, "not-found", nil)
		require.Error(t, err)
		require.NotNil(t, resp.JSON404)
		assert.Equal(t, "NOT_FOUND", *resp.JSON404.Code)
		assert.Equal(t, "Item not found", *resp.JSON404.Message)
	})
}

func TestE2EStrictServer(t *testing.T) {
	e := echo.New()
	handler := &StrictEchoHandler{}
	strict.RegisterStrictHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	client := strict.NewClient(server.URL)
	ctx := context.Background()

	t.Run("JSON round-trip", func(t *testing.T) {
		msg := "hello strict"
		num := 99
		nestedVal := "strict-nested"
		resp, err := client.EchoJSON(ctx, strict.EchoPayload{
			Message: msg,
			Number:  &num,
			Nested: strict.EchoPayloadNested{
				Value: &nestedVal,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, msg, resp.JSON200.Message)
		assert.Equal(t, num, *resp.JSON200.Number)
		assert.Equal(t, nestedVal, *resp.JSON200.Nested.Value)
	})

	t.Run("Path and query params", func(t *testing.T) {
		filter := "pending"
		resp, err := client.GetItem(ctx, "strict-item-456", &strict.GetItemParams{
			Filter: &filter,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "strict-item-456", *resp.JSON200.ID)
		assert.Equal(t, "pending", *resp.JSON200.Filter)
	})

	t.Run("404 error response", func(t *testing.T) {
		resp, err := client.GetItem(ctx, "not-found", nil)
		require.Error(t, err)
		require.NotNil(t, resp.JSON404)
		assert.Equal(t, "NOT_FOUND", *resp.JSON404.Code)
		assert.Equal(t, "Item not found", *resp.JSON404.Message)
	})
}

func TestE2EHeaderParams(t *testing.T) {
	e := echo.New()
	handler := &StrictEchoHandler{}
	strict.RegisterStrictHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	ctx := context.Background()

	t.Run("Header parameter extraction", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/items/header-test?filter=test", nil)
		require.NoError(t, err)
		req.Header.Set("X-Request-ID", "req-12345")
		req.Header.Set("Accept", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), `"id":"header-test"`)
		assert.Contains(t, string(body), `"filter":"test"`)
		assert.Contains(t, string(body), `"requestId":"req-12345"`)
	})
}

func TestE2EChiServer(t *testing.T) {
	handler := &ChiHandler{}
	router := chiGen.Handler(handler)

	server := httptest.NewServer(router)
	defer server.Close()

	client := chiGen.NewClient(server.URL)
	ctx := context.Background()

	t.Run("JSON round-trip", func(t *testing.T) {
		msg := "hello chi"
		num := 42
		nestedVal := "chi-nested"
		resp, err := client.EchoJSON(ctx, chiGen.EchoPayload{
			Message: msg,
			Number:  &num,
			Nested: chiGen.EchoPayloadNested{
				Value: &nestedVal,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, msg, resp.JSON200.Message)
		assert.Equal(t, num, *resp.JSON200.Number)
		assert.Equal(t, nestedVal, *resp.JSON200.Nested.Value)
	})

	t.Run("Form round-trip", func(t *testing.T) {
		resp, err := client.EchoForm(ctx, chiGen.EchoFormRequest{
			Field1: "chi-field",
			Field2: "456",
			Tags:   []string{"chi-tag1", "chi-tag2"},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "chi-field", *resp.JSON200.ReceivedField1)
		assert.Equal(t, 456, *resp.JSON200.ReceivedField2)
		assert.Equal(t, []string{"chi-tag1", "chi-tag2"}, resp.JSON200.ReceivedTags)
	})

	t.Run("Multipart round-trip", func(t *testing.T) {
		fileContent := []byte("chi file content")
		resp, err := client.EchoMultipart(ctx, chiGen.EchoMultipartRequest{
			File: &chiGen.FileUpload{
				Reader:   bytes.NewReader(fileContent),
				Filename: "chi-test.txt",
			},
			Description: "chi description",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "chi-test.txt", *resp.JSON200.Filename)
		assert.Equal(t, len(fileContent), *resp.JSON200.Size)
		assert.Equal(t, "chi description", *resp.JSON200.Description)
	})

	t.Run("Path and query params", func(t *testing.T) {
		filter := "chi-filter"
		resp, err := client.GetItem(ctx, "chi-item-789", &chiGen.GetItemParams{
			Filter: &filter,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "chi-item-789", *resp.JSON200.ID)
		assert.Equal(t, "chi-filter", *resp.JSON200.Filter)
	})

	t.Run("404 error response", func(t *testing.T) {
		resp, err := client.GetItem(ctx, "not-found", nil)
		require.Error(t, err)
		require.NotNil(t, resp.JSON404)
		assert.Equal(t, "NOT_FOUND", *resp.JSON404.Code)
	})

	t.Run("Cookie params", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/session", nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "chi-session-abc"})

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"sessionId":"chi-session-abc"`)
	})
}

func TestE2EStdlibServer(t *testing.T) {
	handler := &StdlibHandler{}
	router := stdlibGen.Handler(handler)

	server := httptest.NewServer(router)
	defer server.Close()

	client := stdlibGen.NewClient(server.URL)
	ctx := context.Background()

	t.Run("JSON round-trip", func(t *testing.T) {
		msg := "hello stdlib"
		num := 42
		nestedVal := "stdlib-nested"
		resp, err := client.EchoJSON(ctx, stdlibGen.EchoPayload{
			Message: msg,
			Number:  &num,
			Nested: stdlibGen.EchoPayloadNested{
				Value: &nestedVal,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, msg, resp.JSON200.Message)
		assert.Equal(t, num, *resp.JSON200.Number)
		assert.Equal(t, nestedVal, *resp.JSON200.Nested.Value)
	})

	t.Run("Form round-trip", func(t *testing.T) {
		resp, err := client.EchoForm(ctx, stdlibGen.EchoFormRequest{
			Field1: "stdlib-field",
			Field2: "789",
			Tags:   []string{"stdlib-tag1", "stdlib-tag2"},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "stdlib-field", *resp.JSON200.ReceivedField1)
		assert.Equal(t, 789, *resp.JSON200.ReceivedField2)
		assert.Equal(t, []string{"stdlib-tag1", "stdlib-tag2"}, resp.JSON200.ReceivedTags)
	})

	t.Run("Multipart round-trip", func(t *testing.T) {
		fileContent := []byte("stdlib file content")
		resp, err := client.EchoMultipart(ctx, stdlibGen.EchoMultipartRequest{
			File: &stdlibGen.FileUpload{
				Reader:   bytes.NewReader(fileContent),
				Filename: "stdlib-test.txt",
			},
			Description: "stdlib description",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "stdlib-test.txt", *resp.JSON200.Filename)
		assert.Equal(t, len(fileContent), *resp.JSON200.Size)
		assert.Equal(t, "stdlib description", *resp.JSON200.Description)
	})

	t.Run("Path and query params", func(t *testing.T) {
		filter := "stdlib-filter"
		resp, err := client.GetItem(ctx, "stdlib-item-101", &stdlibGen.GetItemParams{
			Filter: &filter,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, "stdlib-item-101", *resp.JSON200.ID)
		assert.Equal(t, "stdlib-filter", *resp.JSON200.Filter)
	})

	t.Run("404 error response", func(t *testing.T) {
		resp, err := client.GetItem(ctx, "not-found", nil)
		require.Error(t, err)
		require.NotNil(t, resp.JSON404)
		assert.Equal(t, "NOT_FOUND", *resp.JSON404.Code)
	})

	t.Run("Cookie params", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/session", nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "stdlib-session-xyz"})

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), `"sessionId":"stdlib-session-xyz"`)
	})
}

func TestE2EMultipleStatusCodes(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	client := basic.NewClient(server.URL)
	ctx := context.Background()

	t.Run("201 Created response", func(t *testing.T) {
		status := basic.StatusActive
		resp, err := client.CreateResource(ctx, basic.NewResource{
			Name:   "new-resource",
			Status: &status,
		})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
		require.NotNil(t, resp.JSON201)
		assert.Equal(t, "res-123", *resp.JSON201.ID)
		assert.Equal(t, "new-resource", *resp.JSON201.Name)
		assert.Equal(t, basic.StatusActive, *resp.JSON201.Status)
	})

	t.Run("204 No Content response", func(t *testing.T) {
		resp, err := client.DeleteResource(ctx, "res-123")
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode)
	})
}

func TestE2ECookieParams(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	ctx := context.Background()

	t.Run("Cookie parameter extraction", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/session", nil)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session-123"})

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var session basic.SessionInfo
		err = json.Unmarshal(body, &session)
		require.NoError(t, err)
		assert.Equal(t, "test-session-123", *session.SessionID)
		assert.Equal(t, "user-456", *session.UserID)
	})

	t.Run("Missing cookie returns error", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/session", nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestE2ESecurityAPIKey(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	ctx := context.Background()

	t.Run("Valid API key returns data", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/secure/data", nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", "valid-api-key")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var data basic.SecureData
		err = json.Unmarshal(body, &data)
		require.NoError(t, err)
		assert.Equal(t, "top-secret-data", *data.Secret)
		assert.Equal(t, "admin", *data.AccessLevel)
	})

	t.Run("Invalid API key returns 401", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/secure/data", nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", "invalid-key")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Missing API key returns 401", func(t *testing.T) {
		httpClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/secure/data", nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestE2EEnums(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	client := basic.NewClient(server.URL)
	ctx := context.Background()

	t.Run("Enum round-trip - pending", func(t *testing.T) {
		status := basic.StatusPending
		resp, err := client.CreateResource(ctx, basic.NewResource{
			Name:   "pending-resource",
			Status: &status,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON201)
		assert.Equal(t, basic.StatusPending, *resp.JSON201.Status)
	})

	t.Run("Enum round-trip - active", func(t *testing.T) {
		status := basic.StatusActive
		resp, err := client.CreateResource(ctx, basic.NewResource{
			Name:   "active-resource",
			Status: &status,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON201)
		assert.Equal(t, basic.StatusActive, *resp.JSON201.Status)
	})

	t.Run("Enum round-trip - completed", func(t *testing.T) {
		status := basic.StatusCompleted
		resp, err := client.CreateResource(ctx, basic.NewResource{
			Name:   "completed-resource",
			Status: &status,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON201)
		assert.Equal(t, basic.StatusCompleted, *resp.JSON201.Status)
	})
}

func TestE2ENullable(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	client := basic.NewClient(server.URL)
	ctx := context.Background()

	t.Run("Nullable field with value", func(t *testing.T) {
		desc := "a description"
		status := basic.StatusActive
		resp, err := client.CreateResource(ctx, basic.NewResource{
			Name:        "resource-with-desc",
			Status:      &status,
			Description: &desc,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON201)
		require.NotNil(t, resp.JSON201.Description)
		assert.Equal(t, "a description", *resp.JSON201.Description)
	})

	t.Run("Nullable field absent", func(t *testing.T) {
		status := basic.StatusActive
		resp, err := client.CreateResource(ctx, basic.NewResource{
			Name:   "resource-no-desc",
			Status: &status,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.JSON201)
		assert.Nil(t, resp.JSON201.Description)
	})
}

func TestE2EDiscriminator(t *testing.T) {
	e := echo.New()
	handler := &BasicEchoHandler{}
	basic.RegisterHandlers(e, handler)

	server := httptest.NewServer(e)
	defer server.Close()

	ctx := context.Background()

	t.Run("Circle shape round-trip", func(t *testing.T) {
		httpClient := &http.Client{}

		circleJSON := `{"type":"circle","radius":5.5}`
		req, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/shapes", bytes.NewBufferString(circleJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var shape basic.Shape
		err = json.Unmarshal(body, &shape)
		require.NoError(t, err)
		assert.Equal(t, "circle", shape.Type)

		circle, err := shape.AsCircle()
		require.NoError(t, err)
		assert.Equal(t, "circle", circle.Type)
		assert.Equal(t, 5.5, circle.Radius)
	})

	t.Run("Rectangle shape round-trip", func(t *testing.T) {
		httpClient := &http.Client{}

		rectJSON := `{"type":"rectangle","width":10.0,"height":20.0}`
		req, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/shapes", bytes.NewBufferString(rectJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var shape basic.Shape
		err = json.Unmarshal(body, &shape)
		require.NoError(t, err)
		assert.Equal(t, "rectangle", shape.Type)

		rect, err := shape.AsRectangle()
		require.NoError(t, err)
		assert.Equal(t, "rectangle", rect.Type)
		assert.Equal(t, 10.0, rect.Width)
		assert.Equal(t, 20.0, rect.Height)
	})

	t.Run("AsCircle fails for rectangle", func(t *testing.T) {
		httpClient := &http.Client{}

		rectJSON := `{"type":"rectangle","width":10.0,"height":20.0}`
		req, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/shapes", bytes.NewBufferString(rectJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var shape basic.Shape
		err = json.Unmarshal(body, &shape)
		require.NoError(t, err)

		_, err = shape.AsCircle()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a Circle")
	})
}

// Ensure Chi imports are used
var _ chi.Router
