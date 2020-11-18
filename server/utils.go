package server

import (
	"encoding/json"
	"github.com/speed18/d18-notebook/log"
	"net/http"
	"strings"
)

func min(n1 int, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

func max(n1 int, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func beforeReq(api apiFunc) apiFunc {
	return func(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
		log.Logger.WithField("url", req.URL).Info("incoming request")
		return api(resp, req)
	}
}

func afterReq(api apiFunc) apiFunc {
	return func(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
		data, status, err := api(resp, req)
		log.Logger.WithField("url", req.URL).Info("done processing request")
		return data, status, err
	}
}

func checkMethod(api apiFunc, method string) apiFunc {
	return func(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
		if strings.ToUpper(method) != req.Method {
			resp.WriteHeader(http.StatusMethodNotAllowed)
			return nil, methodNotAllowError, methodNotAllowErr
		}
		return api(resp, req)
	}
}

func checkAuth(api apiFunc) apiFunc {
	return func(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
		tokenCookie, err := req.Cookie(tokenName)
		if err != nil {
			return nil, notAuthError, notAuthErr
		}
		if isAuth(tokenCookie.Value) {
			return api(resp, req)
		}
		return nil, notAuthError, notAuthErr
	}
}

func makeHandler(api apiFunc) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		ret, status, err := api(resp, req)
		encoder := json.NewEncoder(resp)

		if status != noError {
			log.Logger.WithField("status", status).WithField("err", err).Error("api status != 0")
			_ = encoder.Encode(RespObj{Status: status, Data: nil})
			return
		}

		resp.Header().Set("content-type", "application-json")
		if err := encoder.Encode(RespObj{Status: noError, Data: ret}); err != nil {
			log.Logger.WithField("err", err).Error("encode api ret failed.")
			_ = encoder.Encode(RespObj{Status: encodeError, Data: nil})
		}
	}
}
