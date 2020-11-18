package server

import (
	"encoding/json"
	"net/http"
)

func notesAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var reqIns notesReqObj
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reqIns); err != nil {
		return nil, decodeError, err
	}

	var token string
	if tokenCookie, err := req.Cookie(tokenName); err != nil {
		token = ""
	} else {
		token = tokenCookie.Value
	}

	notesInsPtr, page, err := getNotes(reqIns.PageNo, reqIns.Tag, token)
	if err != nil {
		return nil, getNotesError, err
	}

	return notesRespObj{Notes: notesInsPtr, Page: page}, noError, nil
}

func noteAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var reqIns noteReqObj
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reqIns); err != nil {
		return nil, decodeError, err
	}

	var token string
	if tokenCookie, err := req.Cookie(tokenName); err != nil {
		token = ""
	} else {
		token = tokenCookie.Value
	}

	noteInsPtr, err := getNote(reqIns.NoteID, token)
	if err != nil {
		return nil, getNoteError, err
	}

	return noteRespObj{Note: noteInsPtr}, noError, nil
}

func notePublishAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var reqIns notePublishReqObj
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reqIns); err != nil {
		return nil, decodeError, err
	}

	if reqIns.Title == "" || reqIns.Content == "" || len(reqIns.Tags) <= 0 {
		return nil, paramsError, paramsErr
	}

	noteID, err := publishNote(reqIns.Title, reqIns.Content, reqIns.Private, reqIns.Tags...)
	if err != nil {
		return nil, publishNoteError, err
	}

	return notePublishRespObj{NoteID: noteID}, noError, nil
}

func noteUpdateAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var reqIns noteUpdateReqObj
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reqIns); err != nil {
		return nil, decodeError, err
	}

	if err := updateNote(reqIns.NoteID, reqIns.Title, reqIns.Content, reqIns.Private, reqIns.Tags...); err != nil {
		return nil, updateNoteError, err
	}

	return noteUpdateRespObj{NoteID: reqIns.NoteID}, noError, nil
}

func noteDeleteAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var reqIns noteDeleteReqObj
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reqIns); err != nil {
		return nil, decodeError, err
	}

	if err := deleteNote(reqIns.NoteID); err != nil {
		return nil, deleteNoteError, err
	}

	return noteDeleteRespObj{NoteID: reqIns.NoteID}, noError, nil
}

func tagsAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	tagsIns, err := getTags()
	if err != nil {
		return nil, getTagsError, err
	}
	return tagsRespObj{Tags: tagsIns}, noError, nil
}

func authAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var reqIns authReqObj
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reqIns); err != nil {
		return nil, decodeError, err
	}

	token, err := auth(reqIns.Password)
	if err != nil {
		return nil, authError, err
	}

	cookie := &http.Cookie{Name: tokenName, Value: token, MaxAge: tokenExpire}
	http.SetCookie(resp, cookie)

	return authRespObj{Token: token}, noError, nil
}

func isAuthAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	var token string
	if tokenCookie, err := req.Cookie(tokenName); err != nil {
		token = ""
	} else {
		token = tokenCookie.Value
	}
	return isAuthRespObj{IsAuth: isAuth(token)}, noError, nil
}

func logoutAPI(resp http.ResponseWriter, req *http.Request) (interface{}, int, error) {
	if err := deleteToken(); err != nil {
		return nil, logoutError, err
	}
	cookie := &http.Cookie{Name: tokenName, Value: "", MaxAge: -1}
	http.SetCookie(resp, cookie)
	return nil, noError, nil
}

// ------------------------------------------------------------------

var NotePublishHandler = makeHandler(checkMethod(afterReq(beforeReq(checkAuth(notePublishAPI))), post))
var NotesHandler = makeHandler(checkMethod(afterReq(beforeReq(notesAPI)), post))
var NoteHandler = makeHandler(checkMethod(afterReq(beforeReq(noteAPI)), post))
var NoteUpdateHandler = makeHandler(checkMethod(afterReq(beforeReq(checkAuth(noteUpdateAPI))), post))
var NoteDeleteHandler = makeHandler(checkMethod(afterReq(beforeReq(checkAuth(noteDeleteAPI))), post))
var TagsHandler = makeHandler(checkMethod(afterReq(beforeReq(tagsAPI)), post))
var AuthHandler = makeHandler(checkMethod(afterReq(beforeReq(authAPI)), post))
var IsAuthHandler = makeHandler(checkMethod(afterReq(beforeReq(isAuthAPI)), post))
var LogoutHandler = makeHandler(checkMethod(afterReq(beforeReq(checkAuth(logoutAPI))), post))
