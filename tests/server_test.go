package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	serverChi "github.com/kolah/eugene/v2/tests/generated/server_chi"
	serverEcho "github.com/kolah/eugene/v2/tests/generated/server_echo"
	serverStdlib "github.com/kolah/eugene/v2/tests/generated/server_stdlib"
	strictChi "github.com/kolah/eugene/v2/tests/generated/strict_chi"
	strictEcho "github.com/kolah/eugene/v2/tests/generated/strict_echo"
	strictStdlib "github.com/kolah/eugene/v2/tests/generated/strict_stdlib"
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

type enumEchoHandler struct {
	serverEcho.ServerInterface
	kind  serverEcho.Kind
	scope serverEcho.LinkScope
	list  serverEcho.ListLegacyLinksQueryParams
}

func (h *enumEchoHandler) ListLegacyLinks(ctx echo.Context, params serverEcho.ListLegacyLinksQueryParams) error {
	h.list = params
	return ctx.NoContent(http.StatusNoContent)
}

func (h *enumEchoHandler) GetLegacyLink(ctx echo.Context, kind serverEcho.Kind, scope serverEcho.LinkScope) error {
	h.kind, h.scope = kind, scope
	return ctx.NoContent(http.StatusNoContent)
}

type enumChiHandler struct {
	serverChi.ServerInterface
	kind  serverChi.Kind
	scope serverChi.LinkScope
	list  serverChi.ListLegacyLinksQueryParams
}

func (h *enumChiHandler) ListLegacyLinks(w http.ResponseWriter, _ *http.Request, params serverChi.ListLegacyLinksQueryParams) {
	h.list = params
	w.WriteHeader(http.StatusNoContent)
}

func (h *enumChiHandler) GetLegacyLink(w http.ResponseWriter, _ *http.Request, kind serverChi.Kind, scope serverChi.LinkScope) {
	h.kind, h.scope = kind, scope
	w.WriteHeader(http.StatusNoContent)
}

type enumStdlibHandler struct {
	serverStdlib.ServerInterface
	kind  serverStdlib.Kind
	scope serverStdlib.LinkScope
	list  serverStdlib.ListLegacyLinksQueryParams
}

func (h *enumStdlibHandler) ListLegacyLinks(w http.ResponseWriter, _ *http.Request, params serverStdlib.ListLegacyLinksQueryParams) {
	h.list = params
	w.WriteHeader(http.StatusNoContent)
}

func (h *enumStdlibHandler) GetLegacyLink(w http.ResponseWriter, _ *http.Request, kind serverStdlib.Kind, scope serverStdlib.LinkScope) {
	h.kind, h.scope = kind, scope
	w.WriteHeader(http.StatusNoContent)
}

type enumStrictEchoHandler struct {
	strictEcho.StrictServerInterface
	req  strictEcho.GetLegacyLinkRequestObject
	list strictEcho.ListLegacyLinksRequestObject
}

func (h *enumStrictEchoHandler) ListLegacyLinks(_ context.Context, req strictEcho.ListLegacyLinksRequestObject) (strictEcho.ListLegacyLinksResponseObject, error) {
	h.list = req
	return strictEcho.ListLegacyLinks204Response{}, nil
}

func (h *enumStrictEchoHandler) GetLegacyLink(_ context.Context, req strictEcho.GetLegacyLinkRequestObject) (strictEcho.GetLegacyLinkResponseObject, error) {
	h.req = req
	return strictEcho.GetLegacyLink204Response{}, nil
}

type enumStrictChiHandler struct {
	strictChi.StrictServerInterface
	req  strictChi.GetLegacyLinkRequestObject
	list strictChi.ListLegacyLinksRequestObject
}

func (h *enumStrictChiHandler) ListLegacyLinks(_ context.Context, req strictChi.ListLegacyLinksRequestObject) (strictChi.ListLegacyLinksResponseObject, error) {
	h.list = req
	return strictChi.ListLegacyLinks204Response{}, nil
}

func (h *enumStrictChiHandler) GetLegacyLink(_ context.Context, req strictChi.GetLegacyLinkRequestObject) (strictChi.GetLegacyLinkResponseObject, error) {
	h.req = req
	return strictChi.GetLegacyLink204Response{}, nil
}

type enumStrictStdlibHandler struct {
	strictStdlib.StrictServerInterface
	req  strictStdlib.GetLegacyLinkRequestObject
	list strictStdlib.ListLegacyLinksRequestObject
}

func (h *enumStrictStdlibHandler) ListLegacyLinks(_ context.Context, req strictStdlib.ListLegacyLinksRequestObject) (strictStdlib.ListLegacyLinksResponseObject, error) {
	h.list = req
	return strictStdlib.ListLegacyLinks204Response{}, nil
}

func (h *enumStrictStdlibHandler) GetLegacyLink(_ context.Context, req strictStdlib.GetLegacyLinkRequestObject) (strictStdlib.GetLegacyLinkResponseObject, error) {
	h.req = req
	return strictStdlib.GetLegacyLink204Response{}, nil
}

func TestEnumPathParams(t *testing.T) {
	echoH := &enumEchoHandler{}
	echoServer := echo.New()
	serverEcho.RegisterHandlers(echoServer, echoH)

	chiH := &enumChiHandler{}
	stdlibH := &enumStdlibHandler{}

	strictEchoH := &enumStrictEchoHandler{}
	strictEchoServer := echo.New()
	strictEcho.RegisterStrictHandlers(strictEchoServer, strictEchoH)

	strictChiH := &enumStrictChiHandler{}
	strictChiRouter := chi.NewRouter()
	strictChi.RegisterStrictHandlers(strictChiRouter, strictChiH)

	strictStdlibH := &enumStrictStdlibHandler{}
	strictStdlibMux := http.NewServeMux()
	strictStdlib.RegisterStrictHandlers(strictStdlibMux, strictStdlibH)

	tests := []struct {
		name    string
		handler http.Handler
		check   func(t *testing.T)
	}{
		{"echo", echoServer, func(t *testing.T) {
			require.Equal(t, serverEcho.KindFolder, echoH.kind)
			require.Equal(t, serverEcho.LinkScopePrivate, echoH.scope)
		}},
		{"chi", serverChi.Handler(chiH), func(t *testing.T) {
			require.Equal(t, serverChi.KindFolder, chiH.kind)
			require.Equal(t, serverChi.LinkScopePrivate, chiH.scope)
		}},
		{"stdlib", serverStdlib.Handler(stdlibH), func(t *testing.T) {
			require.Equal(t, serverStdlib.KindFolder, stdlibH.kind)
			require.Equal(t, serverStdlib.LinkScopePrivate, stdlibH.scope)
		}},
		{"strict echo", strictEchoServer, func(t *testing.T) {
			require.Equal(t, strictEcho.KindFolder, strictEchoH.req.Kind)
			require.Equal(t, strictEcho.LinkScopePrivate, strictEchoH.req.Scope)
		}},
		{"strict chi", strictChiRouter, func(t *testing.T) {
			require.Equal(t, strictChi.KindFolder, strictChiH.req.Kind)
			require.Equal(t, strictChi.LinkScopePrivate, strictChiH.req.Scope)
		}},
		{"strict stdlib", strictStdlibMux, func(t *testing.T) {
			require.Equal(t, strictStdlib.KindFolder, strictStdlibH.req.Kind)
			require.Equal(t, strictStdlib.LinkScopePrivate, strictStdlibH.req.Scope)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/legacy-links/folder/private", nil))
			require.Equal(t, http.StatusNoContent, rec.Code)
			tt.check(t)

			for _, path := range []string{"/legacy-links/page/private", "/legacy-links/folder/secret"} {
				rec := httptest.NewRecorder()
				tt.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
				require.Equal(t, http.StatusBadRequest, rec.Code, path)
			}
		})
	}
}

func TestEnumArrayQueryParams(t *testing.T) {
	echoH := &enumEchoHandler{}
	echoServer := echo.New()
	serverEcho.RegisterHandlers(echoServer, echoH)

	chiH := &enumChiHandler{}
	stdlibH := &enumStdlibHandler{}

	strictEchoH := &enumStrictEchoHandler{}
	strictEchoServer := echo.New()
	strictEcho.RegisterStrictHandlers(strictEchoServer, strictEchoH)

	strictChiH := &enumStrictChiHandler{}
	strictChiRouter := chi.NewRouter()
	strictChi.RegisterStrictHandlers(strictChiRouter, strictChiH)

	strictStdlibH := &enumStrictStdlibHandler{}
	strictStdlibMux := http.NewServeMux()
	strictStdlib.RegisterStrictHandlers(strictStdlibMux, strictStdlibH)

	kinds := []string{"post", "folder"}
	tests := []struct {
		name    string
		handler http.Handler
		check   func(t *testing.T)
	}{
		{"echo", echoServer, func(t *testing.T) {
			require.Equal(t, kinds, echoH.list.Kinds)
			require.Equal(t, &[]serverEcho.LinkScope{serverEcho.LinkScopePublic, serverEcho.LinkScopePrivate}, echoH.list.Scopes)
		}},
		{"chi", serverChi.Handler(chiH), func(t *testing.T) {
			require.Equal(t, kinds, chiH.list.Kinds)
			require.Equal(t, &[]serverChi.LinkScope{serverChi.LinkScopePublic, serverChi.LinkScopePrivate}, chiH.list.Scopes)
		}},
		{"stdlib", serverStdlib.Handler(stdlibH), func(t *testing.T) {
			require.Equal(t, kinds, stdlibH.list.Kinds)
			require.Equal(t, &[]serverStdlib.LinkScope{serverStdlib.LinkScopePublic, serverStdlib.LinkScopePrivate}, stdlibH.list.Scopes)
		}},
		{"strict echo", strictEchoServer, func(t *testing.T) {
			require.Equal(t, kinds, strictEchoH.list.Kinds)
			require.Equal(t, &[]strictEcho.LinkScope{strictEcho.LinkScopePublic, strictEcho.LinkScopePrivate}, strictEchoH.list.Scopes)
		}},
		{"strict chi", strictChiRouter, func(t *testing.T) {
			require.Equal(t, kinds, strictChiH.list.Kinds)
			require.Equal(t, &[]strictChi.LinkScope{strictChi.LinkScopePublic, strictChi.LinkScopePrivate}, strictChiH.list.Scopes)
		}},
		{"strict stdlib", strictStdlibMux, func(t *testing.T) {
			require.Equal(t, kinds, strictStdlibH.list.Kinds)
			require.Equal(t, &[]strictStdlib.LinkScope{strictStdlib.LinkScopePublic, strictStdlib.LinkScopePrivate}, strictStdlibH.list.Scopes)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/legacy-links?kinds=post&kinds=folder&scopes=public&scopes=private", nil))
			require.Equal(t, http.StatusNoContent, rec.Code)
			tt.check(t)

			rec = httptest.NewRecorder()
			tt.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/legacy-links?kinds=post&scopes=public&scopes=secret", nil))
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
