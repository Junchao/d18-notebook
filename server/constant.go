package server

import (
	"errors"
)

const get = "get"

const post = "post"

const tokenName = "token"

const tokenKey = "notebook:token"

const tokenExpire = 3600 * 24 * 3

// ------------------------------------------------------------------

const defaultError = -1

const noError = 0

const notAuthError = -1000

const authError = -1001

const logoutError = -1002

const encodeError = -1003

const decodeError = -1004

const paramsError = -1005

const methodNotAllowError = -1006

const publishNoteError = -2000

const getNotesError = -2001

const getNoteError = -2002

const updateNoteError = -2003

const deleteNoteError = -2004

const getTagsError = -2010

// ------------------------------------------------------------------

var methodNotAllowErr = errors.New("method not allow")

var notAuthErr = errors.New("not auth")

var paramsErr = errors.New("params error")

var noteNotExistsErr = errors.New("notes does not exists")
