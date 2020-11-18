package server

import (
	"database/sql"
	"net/http"
)

type apiFunc func(resp http.ResponseWriter, req *http.Request) (interface{}, int, error)

type txFunc func(cursor cursorObj) (interface{}, error)

type cursorObj interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type noteObj struct {
	ID        uint32   `json:"id"`
	Title     string   `json:"title"`
	Author    string   `json:"author"`
	Content   string   `json:"content"`
	PlainText string   `json:"plain_text"`
	Private   bool     `json:"private"`
	Words     uint32   `json:"words"`
	Tags      []tagObj `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdateAt  string   `json:"update_at"`
}

type notesInsPtrSlice []*noteObj

func (ns notesInsPtrSlice) Len() int {
	return len(ns)
}

func (ns notesInsPtrSlice) Less(i int, j int) bool {
	return ns[i].UpdateAt > ns[j].UpdateAt
}

func (ns notesInsPtrSlice) Swap(i int, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

type tagObj struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Tagged uint32 `json:"tagged"`
}

type pageObj struct {
	Left  uint32 `json:"left"`
	Right uint32 `json:"right"`
	Cur   uint32 `json:"cur"`
	Total uint32 `json:"total"`
}

type noteSqlObj struct {
	ID        uint32
	Title     string
	Author    string
	Content   string
	PlainText string
	Auth      bool
	Words     uint32
	CreatedAt string
	UpdatedAt string
	TagID     sql.NullInt64
	TagName   sql.NullString
}

type notePublishReqObj struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Private bool     `json:"private"`
}

type noteUpdateReqObj struct {
	NoteID  uint32   `json:"note_id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Private bool     `json:"private"`
}

type notesReqObj struct {
	PageNo uint32 `json:"page_no"`
	Tag    uint32 `json:"tag"`
}

type noteReqObj struct {
	NoteID uint32 `json:"note_id"`
}

type noteDeleteReqObj struct {
	NoteID uint32 `json:"note_id"`
}

type authReqObj struct {
	Password string `json:"password"`
}

type IsAuthReqObj struct {
	Token string `json:"token"`
}

type RespObj struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
}

type notePublishRespObj struct {
	NoteID uint32 `json:"note_id"`
}

type noteUpdateRespObj struct {
	NoteID uint32 `json:"note_id"`
}

type noteDeleteRespObj struct {
	NoteID uint32 `json:"note_id"`
}

type notesRespObj struct {
	Notes []*noteObj `json:"notes"`
	Page  pageObj    `json:"page"`
}

type noteRespObj struct {
	Note *noteObj `json:"note"`
}

type tagsRespObj struct {
	Tags []tagObj `json:"tags"`
}

type authRespObj struct {
	Token string `json:"token"`
}

type isAuthRespObj struct {
	IsAuth bool `json:"is_auth"`
}
