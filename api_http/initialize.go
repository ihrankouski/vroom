package api_http

import (
	"fmt"
	"net/http"
	"path"

	"github.com/deze333/reseer"
	"github.com/deze333/vroom/api_ws"
	"github.com/deze333/vroom/api_xhr"
	"github.com/deze333/vroom/auth"
	"github.com/deze333/vroom/reqres"
	"github.com/deze333/vroom/util"
	"github.com/gorilla/mux"
)

//------------------------------------------------------------
// Initialize
//------------------------------------------------------------

var _ctx Ctx
var _versionWatcher *reseer.Seer

//------------------------------------------------------------
// Initialize
//------------------------------------------------------------

// Initializes routers.
func Initialize(ctx Ctx) (err error) {

	// Authentication
	if err = Initialize_Auth(&ctx); err != nil {
		return
	}

	// Directories
	if err = Initialize_Dirs(ctx.Dirs); err != nil {
		return
	}

	// On Panic
	if ctx.OnPanic == nil {
		return fmt.Errorf("OnPanic handler must be provided")
	}

	// ReqRes
	if err = Initialize_ReqRes(&ctx); err != nil {
		return
	}

	// XHR
	if err = Initialize_XHR(&ctx); err != nil {
		return
	}

	// Websocket
	if err = Initialize_WS(&ctx); err != nil {
		return
	}

	// Router
	if err = Initialize_Router(&ctx); err != nil {
		return
	}

	// Success, remember context
	_ctx = ctx
	return
}

// Initializes authentication mechanism.
func Initialize_Auth(ctx *Ctx) (err error) {

	params := ctx.Auth

	if params.CookieName == "" {
		return fmt.Errorf("Cookie name must be specified")
	}

	if params.CookieStoreId == "" {
		return fmt.Errorf("Cookie store ID must be specified")
	}

	err = auth.Initialize(
		params.CookieName,
		params.CookieStoreId,
		params.CookiePath,
		params.CookieDomain,
		params.CookieMaxAge,
		ctx.OnPanic,
	)

	// Register de-auth listeners, ie persistent connections
	auth.AddListener_DeAuth(api_ws.CloseAuthdConn)

	return
}

// Initializes directories.
func Initialize_Dirs(dirs Dirs) (err error) {

	// Start watcher that will report
	// changes to application files
	_versionWatcher, err = reseer.New(
		path.Join(dirs.VersionFileDir, "reseer_track.csv"),
		dirs.AppWatchNotify,
		onAppVersionChanged)

	// Set initial version
	if err == nil {
		setAppVersion(_versionWatcher.VersionTxt())
	}

	return
}

// Initializes ReqRes package.
func Initialize_ReqRes(ctx *Ctx) (err error) {

	err = reqres.Initialize(ctx.OnPanic)
	return
}

// Initializes Websocket package.
func Initialize_WS(ctx *Ctx) (err error) {

	err = api_ws.Initialize(GetVersion, ctx.OnPanic)
	return
}

// Initializes XHR package.
func Initialize_XHR(ctx *Ctx) (err error) {

	err = api_xhr.Initialize(ctx.OnPanic)
	return
}

// Initializes router.
func Initialize_Router(ctx *Ctx) (err error) {

	summ := util.NewSummary("Registered HTTP routes", "Type", "Route", "Handler", "Package")

	// Register Gorilla Mux router handlers
	router := mux.NewRouter()

	// ------ HTML ------

	// HTML Public routes
	for r, h := range ctx.Handlers_HTML.Public {
		makeRouteHandlers(ctx, router, r, h, false, "HTML Public", summ)
	}
	summ.AddBlankLine()

	// HTML Authd routes
	for r, h := range ctx.Handlers_HTML.Authd {
		makeRouteHandlers(ctx, router, r, h, true, "HTML Authd", summ)
	}
	summ.AddBlankLine()

	// ------ XHR ------

	// XHR Public routes
	for r, h := range ctx.Handlers_XHR.Public {
		addSumm(summ, "XHR Public", r, h)
		router.HandleFunc(r, makeHandler_XHR(ctx, h, false))
	}
	summ.AddBlankLine()

	// XHR Authrd routes
	for r, h := range ctx.Handlers_XHR.Authd {
		addSumm(summ, "XHR Authd", r, h)
		router.HandleFunc(r, makeHandler_XHR(ctx, h, true))
	}
	summ.AddBlankLine()

	// ------ WS ------

	// WS Public routes
	for _, wsr := range ctx.Handlers_WS.Public {
		addSumm(summ, "WS Public", wsr.URL, wsr.Procs)
		router.HandleFunc(wsr.URL, makeHandler_WS(ctx, &wsr, false))
	}
	summ.AddBlankLine()

	// WS Authrd routes
	for _, wsr := range ctx.Handlers_WS.Authd {
		addSumm(summ, "WS Authd", wsr.URL, wsr.Procs)
		router.HandleFunc(wsr.URL, makeHandler_WS(ctx, &wsr, true))
	}
	summ.AddBlankLine()

	// ------ FILE ------

	// FILE Public routes
	for r, h := range ctx.Handlers_FILE.Public {
		addSumm(summ, "FILE Public", r, h)
		router.HandleFunc(r, makeHandler_HTML(ctx, h, false))
	}
	summ.AddBlankLine()

	// ------ NOT FOUND ------

	// Set default not found router
	notFoundH := ctx.Handlers_HTML.NotFound
	router.NotFoundHandler = makeHandler_NotFound(ctx, notFoundH)
	addSumm(summ, "HTML Not Found", "*", notFoundH)
	summ.AddBlankLine()

	// Set Gorilla Mux as root router
	http.Handle("/", router)

	// Output summary
	fmt.Println(summ)
	return
}

// Makes handlers for given route and its synonyms.
func makeRouteHandlers(ctx *Ctx, router *mux.Router, r string, h H, authd bool, descr string, summ *util.Summary) {
	for _, r := range routeSynonyms(r) {
		router.HandleFunc(r, makeHandler_HTML(ctx, h, authd))
	}
	addSumm(summ, descr, r, h)
}

// Adds summary on the router.
func addSumm(summ *util.Summary, descr, route string, obj interface{}) {
	summ.AddLine(append([]string{descr, route}, util.GetFuncInfo(obj)...)...)
}

// De-initalizes components.
func DeInitialize() {

	// Stop version watcher
	if _versionWatcher != nil {
		_versionWatcher.Stop()
	}
}
