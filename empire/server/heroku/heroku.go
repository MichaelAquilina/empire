package heroku

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/httpx/middleware"
)

// The Accept header that controls the api version. See
// https://devcenter.heroku.com/articles/platform-api-reference#clients
const AcceptHeader = "application/vnd.heroku+json; version=3"

// New creates the API routes and returns a new http.Handler to serve them.
func New(e *empire.Empire, auth Authorizer) http.Handler {
	r := httpx.NewRouter()

	// Apps
	r.Handle("GET", "/apps", Authenticate(e, &GetApps{e}))                 // hk apps
	r.Handle("DELETE", "/apps/{app}", Authenticate(e, &DeleteApp{e}))      // hk destroy
	r.Handle("POST", "/apps", Authenticate(e, &PostApps{e}))               // hk create
	r.Handle("POST", "/organizations/apps", Authenticate(e, &PostApps{e})) // hk create

	// Deploys
	r.Handle("POST", "/deploys", Authenticate(e, &PostDeploys{e})) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", Authenticate(e, &GetReleases{e}))          // hk releases
	r.Handle("GET", "/apps/{app}/releases/{version}", Authenticate(e, &GetRelease{e})) // hk release-info
	r.Handle("POST", "/apps/{app}/releases", Authenticate(e, &PostReleases{e}))        // hk rollback

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", Authenticate(e, &GetConfigs{e}))     // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", Authenticate(e, &PatchConfigs{e})) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", Authenticate(e, &GetProcesses{e})) // hk dynos

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", Authenticate(e, &PatchFormation{e})) // hk scale

	// OAuth
	r.Handle("POST", "/oauth/authorizations", &PostAuthorizations{e, auth})

	// Wrap the router in middleware to handle errors.
	h := middleware.HandleError(r, func(err error, w http.ResponseWriter, r *http.Request) {
		Error(w, err, http.StatusInternalServerError)
	})

	// Wrap the route in middleware to add a context.Context.
	b := middleware.BackgroundContext(h)

	return b
}

// Encode json ecnodes v into w.
func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}

// Decode json decodes the request body into v.
func Decode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Error is used to respond with errors in the heroku error format, which is
// specified at
// https://devcenter.heroku.com/articles/platform-api-reference#errors
//
// If an ErrorResource is provided as the error, and it provides a non-zero
// status, that will be used as the response status code.
func Error(w http.ResponseWriter, err error, status int) error {
	var v interface{}
	switch err := err.(type) {
	case *ErrorResource:
		if err.Status != 0 {
			status = err.Status
		}

		v = err
	case *empire.ValidationError:
		v = ErrBadRequest
	default:
		v = &ErrorResource{
			Message: err.Error(),
		}
	}

	log.Printf("error=%+v\n", v)
	w.WriteHeader(status)
	return Encode(w, v)
}

// NoContent responds with a 404 and an empty body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return Encode(w, nil)
}