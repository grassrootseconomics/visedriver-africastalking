package at

import (
	"io"
	"net/http"
	"strconv"

	"git.defalsify.org/vise.git/logging"
	"git.grassecon.net/grassrootseconomics/visedriver/request"
	"git.grassecon.net/grassrootseconomics/visedriver/errors"
)

var (
	logg = logging.NewVanilla().WithDomain("atserver").WithContextKey("at-session-id").WithContextKey("SessionId")
)

type ATRequestHandler struct {
	//*httpsession.RequestHandler
	request.RequestHandler
}

func NewATRequestHandler(h request.RequestHandler) *ATRequestHandler {
	return &ATRequestHandler{
		RequestHandler: h,
	}
}

func (ash *ATRequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var code int
	var err error

	rqs := request.RequestSession{
		Ctx:    req.Context(),
		Writer: w,
	}

	rp := ash.GetRequestParser()
	cfg := ash.GetConfig()
	cfg.SessionId, err = rp.GetSessionId(req.Context(), req)
	if err != nil {
		logg.ErrorCtxf(rqs.Ctx, "", "header processing error", err)
		ash.WriteError(w, 400, err)
		return
	}
	rqs.Config = cfg
	rqs.Input, err = rp.GetInput(req)
	if err != nil {
		logg.ErrorCtxf(rqs.Ctx, "", "header processing error", err)
		ash.WriteError(w, 400, err)
		return
	}

	rqs, err = ash.Process(rqs) 
	switch err {
	case nil: // set code to 200 if no err
		code = 200
	case errors.ErrStorage, errors.ErrEngineInit, errors.ErrEngineExec, errors.ErrEngineType:
		code = 500
	default:
		code = 500
	}

	if code != 200 {
		ash.WriteError(w, 500, err)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain")
	rqs, err = ash.Output(rqs)
	if err != nil {
		ash.WriteError(w, 500, err)
		return
	}

	rqs, err = ash.Reset(rqs.Ctx, rqs)
	if err != nil {
		ash.WriteError(w, 500, err)
		return
	}
}

func (ash *ATRequestHandler) Output(rqs request.RequestSession) (request.RequestSession, error) {
	var err error
	var prefix string

	if rqs.Continue {
		prefix = "CON "
	} else {
		prefix = "END "
	}

	_, err = io.WriteString(rqs.Writer, prefix)
	if err != nil {
		return rqs, err
	}

	_, err = rqs.Engine.Flush(rqs.Ctx, rqs.Writer)
	return rqs, err
}

func (ash *ATRequestHandler) WriteError(w http.ResponseWriter, code int, err error) {
	s := err.Error()
	w.Header().Set("Content-Length", strconv.Itoa(len(s)))
	w.WriteHeader(code)
	_, err = w.Write([]byte(s))
	if err != nil {
		logg.Errorf("error writing error!!", "err", err, "olderr", s)
		w.WriteHeader(500)
	}
}
