package tests

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	basic "github.com/kolah/eugene/v2/tests/generated/e2e_echo"
	stdlibGen "github.com/kolah/eugene/v2/tests/generated/e2e_stdlib"
	strict "github.com/kolah/eugene/v2/tests/generated/e2e_strict_echo"
	serverChi "github.com/kolah/eugene/v2/tests/generated/server_chi"
	serverEcho "github.com/kolah/eugene/v2/tests/generated/server_echo"
	serverStdlib "github.com/kolah/eugene/v2/tests/generated/server_stdlib"
	strictChi "github.com/kolah/eugene/v2/tests/generated/strict_chi"
	strictEcho "github.com/kolah/eugene/v2/tests/generated/strict_echo"
	strictStdlib "github.com/kolah/eugene/v2/tests/generated/strict_stdlib"

	chiGen "github.com/kolah/eugene/v2/tests/generated/e2e_chi"
)

// requestError is how a service matches the RequestError of any generated package.
type requestError interface {
	error
	Location() string
	Parameter() string
	Missing() bool
}

// errorRecorder keeps the last error a server reported and answers like the default.
type errorRecorder struct{ err error }

func (rec *errorRecorder) handle(w http.ResponseWriter, _ *http.Request, err error) {
	rec.err = err
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (rec *errorRecorder) echo() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		rec.err = err
		e.DefaultHTTPErrorHandler(err, c)
	}
	return e
}

type recordingServer struct {
	handler http.Handler
	rec     *errorRecorder
}

type requestErrorCase struct {
	name    string
	servers []string // empty means every server
	req     func() *http.Request
	in      string
	param   string
	missing bool
	message string
	cause   error
}

func get(target string, header ...string) func() *http.Request {
	return func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		for i := 0; i+1 < len(header); i += 2 {
			req.Header.Set(header[i], header[i+1])
		}
		return req
	}
}

func post(target, contentType, body string) func() *http.Request {
	return func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		return req
	}
}

func checkRequestErrors(t *testing.T, servers map[string]recordingServer, cases []requestErrorCase) {
	t.Helper()
	for name, s := range servers {
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				if len(tc.servers) > 0 && !slices.Contains(tc.servers, name) {
					continue
				}
				t.Run(tc.name, func(t *testing.T) {
					s.rec.err = nil
					w := httptest.NewRecorder()
					s.handler.ServeHTTP(w, tc.req())
					require.Equal(t, http.StatusBadRequest, w.Code)

					var re requestError
					require.ErrorAs(t, s.rec.err, &re)
					assert.Equal(t, tc.in, re.Location())
					assert.Equal(t, tc.param, re.Parameter())
					assert.Equal(t, tc.missing, re.Missing())
					assert.Equal(t, tc.message, re.Error())
					if tc.cause != nil {
						assert.ErrorIs(t, s.rec.err, tc.cause)
					}
				})
			}
		})
	}
}

func TestRequestErrorsAreTyped(t *testing.T) {
	echoRec, strictRec, chiRec, stdlibRec := &errorRecorder{}, &errorRecorder{}, &errorRecorder{}, &errorRecorder{}
	echoServer := echoRec.echo()
	basic.RegisterHandlers(echoServer, &BasicEchoHandler{})
	strictServer := strictRec.echo()
	strict.RegisterStrictHandlers(strictServer, &StrictEchoHandler{})

	servers := map[string]recordingServer{
		"echo":        {echoServer, echoRec},
		"strict echo": {strictServer, strictRec},
		"chi":         {chiGen.HandlerWithOptions(&ChiHandler{}, chiGen.ChiServerOptions{RequestErrorHandler: chiRec.handle}), chiRec},
		"stdlib":      {stdlibGen.HandlerWithOptions(&StdlibHandler{}, stdlibGen.StdlibServerOptions{RequestErrorHandler: stdlibRec.handle}), stdlibRec},
	}
	plain := []string{"echo", "chi", "stdlib"}

	checkRequestErrors(t, servers, []requestErrorCase{
		{name: "missing header", req: get("/whoami", "X-Tenant-ID", "7"),
			in: "header", param: "X-User-ID", missing: true, message: "missing X-User-ID"},
		{name: "unparseable header", req: get("/whoami", "X-User-ID", "u", "X-Tenant-ID", "seven"),
			in: "header", param: "X-Tenant-ID", message: "invalid X-Tenant-ID", cause: strconv.ErrSyntax},
		{name: "missing query value", req: get("/search?tenant=7&tags=a"),
			in: "query", param: "user", missing: true, message: "missing user"},
		{name: "unparseable query value", req: get("/search?user=u&tenant=seven&tags=a"),
			in: "query", param: "tenant", message: "invalid tenant", cause: strconv.ErrSyntax},
		{name: "invalid query enum", req: get("/search?user=u&tenant=7&tags=a&status=bogus"),
			in: "query", param: "status", message: "invalid status"},
		{name: "missing query array", req: get("/search?user=u&tenant=7"),
			in: "query", param: "tags", missing: true, message: "missing tags"},
		{name: "missing cookie", req: get("/session"),
			in: "cookie", param: "session_id", missing: true, message: "missing session_id"},
		{name: "unparseable multipart body", servers: plain, req: post("/echo/multipart", "multipart/form-data", "x"),
			in: "body", message: "invalid request body"},
		{name: "unparseable JSON body", servers: []string{"strict echo"}, req: post("/echo/json", "application/json", "{"),
			in: "body", message: "invalid request body"},
	})
}

func TestRequestErrorsAreTypedForPathAndStrictBodies(t *testing.T) {
	recs := map[string]*errorRecorder{}
	for _, name := range []string{"echo", "chi", "stdlib", "strict echo", "strict chi", "strict stdlib"} {
		recs[name] = &errorRecorder{}
	}
	echoServer := recs["echo"].echo()
	serverEcho.RegisterHandlers(echoServer, &enumEchoHandler{})
	strictEchoServer := recs["strict echo"].echo()
	strictEcho.RegisterStrictHandlers(strictEchoServer, &enumStrictEchoHandler{})
	strictChiRouter := chi.NewRouter()
	strictChi.RegisterStrictHandlers(strictChiRouter, &enumStrictChiHandler{}, strictChi.StrictOptions{RequestErrorHandler: recs["strict chi"].handle})
	strictStdlibMux := http.NewServeMux()
	strictStdlib.RegisterStrictHandlers(strictStdlibMux, &enumStrictStdlibHandler{}, strictStdlib.StrictOptions{RequestErrorHandler: recs["strict stdlib"].handle})

	servers := map[string]recordingServer{
		"echo":          {echoServer, recs["echo"]},
		"chi":           {serverChi.HandlerWithOptions(&enumChiHandler{}, serverChi.ChiServerOptions{RequestErrorHandler: recs["chi"].handle}), recs["chi"]},
		"stdlib":        {serverStdlib.HandlerWithOptions(&enumStdlibHandler{}, serverStdlib.StdlibServerOptions{RequestErrorHandler: recs["stdlib"].handle}), recs["stdlib"]},
		"strict echo":   {strictEchoServer, recs["strict echo"]},
		"strict chi":    {strictChiRouter, recs["strict chi"]},
		"strict stdlib": {strictStdlibMux, recs["strict stdlib"]},
	}

	checkRequestErrors(t, servers, []requestErrorCase{
		{name: "invalid path enum", req: get("/legacy-links/page/private"),
			in: "path", param: "kind", message: "invalid kind"},
		{name: "invalid query enum array", req: get("/legacy-links?kinds=post&scopes=public&scopes=bogus"),
			in: "query", param: "scopes", message: "invalid scopes"},
		{name: "missing query array", req: get("/legacy-links?scopes=public"),
			in: "query", param: "kinds", missing: true, message: "missing kinds"},
		{name: "unparseable JSON body", servers: []string{"strict echo", "strict chi", "strict stdlib"},
			req: post("/items", "application/json", `{"name": nope}`), in: "body", message: "invalid request body"},
	})
}

func TestRequestErrorDefaultResponses(t *testing.T) {
	echoServer := echo.New()
	basic.RegisterHandlers(echoServer, &BasicEchoHandler{})
	strictServer := echo.New()
	strict.RegisterStrictHandlers(strictServer, &StrictEchoHandler{})
	strictChiRouter := chi.NewRouter()
	strictChi.RegisterStrictHandlers(strictChiRouter, &enumStrictChiHandler{})
	strictStdlibMux := http.NewServeMux()
	strictStdlib.RegisterStrictHandlers(strictStdlibMux, &enumStrictStdlibHandler{})

	tests := []struct {
		name    string
		handler http.Handler
		req     func() *http.Request
		body    string
	}{
		{"chi missing header", chiGen.Handler(&ChiHandler{}), get("/whoami", "X-Tenant-ID", "7"), "missing X-User-ID\n"},
		{"stdlib missing header", stdlibGen.Handler(&StdlibHandler{}), get("/whoami", "X-Tenant-ID", "7"), "missing X-User-ID\n"},
		{"echo missing header", echoServer, get("/whoami", "X-Tenant-ID", "7"), `{"message":"missing X-User-ID"}` + "\n"},
		{"strict echo missing header", strictServer, get("/whoami", "X-Tenant-ID", "7"), `{"message":"missing X-User-ID"}` + "\n"},
		{"echo unparseable header", echoServer, get("/whoami", "X-User-ID", "u", "X-Tenant-ID", "seven"), `{"message":"invalid X-Tenant-ID"}` + "\n"},
		{"chi unparseable multipart body", chiGen.Handler(&ChiHandler{}), post("/echo/multipart", "multipart/form-data", "x"), "invalid request body\n"},
		{"echo unparseable multipart body", echoServer, post("/echo/multipart", "multipart/form-data", "x"), `{"message":"invalid request body"}` + "\n"},
		{"strict chi invalid path enum", strictChiRouter, get("/legacy-links/page/private"), "invalid kind\n"},
		{"strict stdlib unparseable body", strictStdlibMux, post("/items", "application/json", "{"), "invalid request body\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, tt.req())
			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, tt.body, w.Body.String())
		})
	}
}
