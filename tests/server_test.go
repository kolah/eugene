package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestEchoServerRouting(t *testing.T) {
	e := echo.New()

	var capturedID string
	var capturedMethod string

	handler := &mockEchoHandler{
		getPetFunc: func(ctx echo.Context) error {
			capturedMethod = ctx.Request().Method
			capturedID = ctx.Param("id")
			return ctx.String(http.StatusOK, "ok")
		},
		uploadPetPhotoFunc: func(ctx echo.Context, id string) error {
			capturedMethod = ctx.Request().Method
			capturedID = id
			return ctx.String(http.StatusOK, "ok")
		},
	}

	registerEchoHandlers(e, handler)

	t.Run("GET /pets/:id extracts path param", func(t *testing.T) {
		capturedID = ""
		req := httptest.NewRequest(http.MethodGet, "/pets/123", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "123", capturedID)
		require.Equal(t, http.MethodGet, capturedMethod)
	})

	t.Run("POST /pets/:id/images extracts path param", func(t *testing.T) {
		capturedID = ""
		req := httptest.NewRequest(http.MethodPost, "/pets/456/images", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "456", capturedID)
		require.Equal(t, http.MethodPost, capturedMethod)
	})

	t.Run("non-existent route returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestChiServerRouting(t *testing.T) {
	r := chi.NewRouter()

	var capturedID string
	var capturedMethod string

	handler := &mockChiHandler{
		getPetFunc: func(w http.ResponseWriter, r *http.Request) {
			capturedMethod = r.Method
			capturedID = chi.URLParam(r, "id")
			w.WriteHeader(http.StatusOK)
		},
		uploadPetPhotoFunc: func(w http.ResponseWriter, r *http.Request, id string) {
			capturedMethod = r.Method
			capturedID = id
			w.WriteHeader(http.StatusOK)
		},
	}

	registerChiHandlers(r, handler)

	t.Run("GET /pets/{id} extracts path param", func(t *testing.T) {
		capturedID = ""
		req := httptest.NewRequest(http.MethodGet, "/pets/abc", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "abc", capturedID)
		require.Equal(t, http.MethodGet, capturedMethod)
	})

	t.Run("POST /pets/{id}/images extracts path param", func(t *testing.T) {
		capturedID = ""
		req := httptest.NewRequest(http.MethodPost, "/pets/xyz/images", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "xyz", capturedID)
		require.Equal(t, http.MethodPost, capturedMethod)
	})
}

func TestStdlibServerRouting(t *testing.T) {
	mux := http.NewServeMux()

	var capturedID string
	var capturedMethod string

	handler := &mockStdlibHandler{
		getPetFunc: func(w http.ResponseWriter, r *http.Request) {
			capturedMethod = r.Method
			capturedID = r.PathValue("id")
			w.WriteHeader(http.StatusOK)
		},
		uploadPetPhotoFunc: func(w http.ResponseWriter, r *http.Request, id string) {
			capturedMethod = r.Method
			capturedID = id
			w.WriteHeader(http.StatusOK)
		},
	}

	registerStdlibHandlers(mux, handler)

	t.Run("GET /pets/{id} extracts path param", func(t *testing.T) {
		capturedID = ""
		req := httptest.NewRequest(http.MethodGet, "/pets/test-id", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "test-id", capturedID)
		require.Equal(t, http.MethodGet, capturedMethod)
	})

	t.Run("POST /pets/{id}/images extracts path param", func(t *testing.T) {
		capturedID = ""
		req := httptest.NewRequest(http.MethodPost, "/pets/img-id/images", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "img-id", capturedID)
		require.Equal(t, http.MethodPost, capturedMethod)
	})
}

// Mock handlers for Echo
type mockEchoHandler struct {
	getPetFunc         func(ctx echo.Context) error
	uploadPetPhotoFunc func(ctx echo.Context, id string) error
}

func (m *mockEchoHandler) GetPet(ctx echo.Context) error {
	if m.getPetFunc != nil {
		return m.getPetFunc(ctx)
	}
	return ctx.NoContent(http.StatusNotImplemented)
}

func (m *mockEchoHandler) UploadPetPhoto(ctx echo.Context, id string) error {
	if m.uploadPetPhotoFunc != nil {
		return m.uploadPetPhotoFunc(ctx, id)
	}
	return ctx.NoContent(http.StatusNotImplemented)
}

func registerEchoHandlers(e *echo.Echo, h *mockEchoHandler) {
	e.GET("/pets/:id", h.GetPet)
	e.POST("/pets/:id/images", func(c echo.Context) error {
		return h.UploadPetPhoto(c, c.Param("id"))
	})
}

// Mock handlers for Chi
type mockChiHandler struct {
	getPetFunc         func(w http.ResponseWriter, r *http.Request)
	uploadPetPhotoFunc func(w http.ResponseWriter, r *http.Request, id string)
}

func registerChiHandlers(r chi.Router, h *mockChiHandler) {
	r.Get("/pets/{id}", h.getPetFunc)
	r.Post("/pets/{id}/images", func(w http.ResponseWriter, r *http.Request) {
		h.uploadPetPhotoFunc(w, r, chi.URLParam(r, "id"))
	})
}

// Mock handlers for stdlib
type mockStdlibHandler struct {
	getPetFunc         func(w http.ResponseWriter, r *http.Request)
	uploadPetPhotoFunc func(w http.ResponseWriter, r *http.Request, id string)
}

func registerStdlibHandlers(mux *http.ServeMux, h *mockStdlibHandler) {
	mux.HandleFunc("GET /pets/{id}", h.getPetFunc)
	mux.HandleFunc("POST /pets/{id}/images", func(w http.ResponseWriter, r *http.Request) {
		h.uploadPetPhotoFunc(w, r, r.PathValue("id"))
	})
}
