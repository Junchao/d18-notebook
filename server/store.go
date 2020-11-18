package server

import (
	"database/sql"
	"fmt"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/speed18/d18-notebook/log"
	"sort"
	"strings"
)

var DB *sql.DB
var RDS *redis.Client

func withTransaction(tf txFunc) (interface{}, error) {
	var err error
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			log.Logger.Error("transaction error, rollback now")
			_ = tx.Rollback()
			return
		}
	}()

	ret, err := tf(tx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return ret, nil
}

// ------------------------------------------------------------------

func insertTags(cursor cursorObj, ignoreDuplicated bool, names ...string) (uint32, error) {
	var params []string
	for i := 0; i < len(names); i++ {
		params = append(params, "(?)")
	}

	var sqlStr string
	if ignoreDuplicated {
		sqlStr = fmt.Sprintf("insert ignore into notebook.tag (name) values %s", strings.Join(params, ","))
	} else {
		sqlStr = fmt.Sprintf("insert into notebook.tag (name) values %s", strings.Join(params, ","))
	}
	log.Logger.WithField("sql", sqlStr).Debug()

	var args []interface{}
	for _, name := range names {
		args = append(args, name)
	}

	result, err := cursor.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}

	tagID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint32(tagID), nil
}

func selectTagsWithNotesCount(cursor cursorObj, closeRows bool) ([]tagObj, error) {
	tagsIns := make([]tagObj, 0)

	sqlStr := `select tag_id, tag_name, count(note_id) as cnt
					from notebook.note_tag
					group by tag_id, tag_name`
	rows, err := cursor.Query(sqlStr)
	if err != nil {
		return tagsIns, err
	}
	if closeRows {
		defer rows.Close()
	}

	for rows.Next() {
		var tagIns tagObj
		err := rows.Scan(&tagIns.ID, &tagIns.Name, &tagIns.Tagged)
		if err != nil {
			return tagsIns, err
		}
		tagsIns = append(tagsIns, tagIns)
	}

	err = rows.Err()
	if err != nil {
		return tagsIns, err
	}

	return tagsIns, nil
}

func selectTagsByName(cursor cursorObj, closeRows bool, forUpdate bool, names ...string) ([]tagObj, error) {
	tagsIns := make([]tagObj, 0)

	var params []string
	for i := 0; i < len(names); i++ {
		params = append(params, "?")
	}
	var sqlStr string
	if forUpdate {
		sqlStr = fmt.Sprintf("select id, name from notebook.tag where name in (%s) for update", strings.Join(params, ","))
	} else {
		sqlStr = fmt.Sprintf("select id, name from notebook.tag where name in (%s)", strings.Join(params, ","))
	}
	log.Logger.WithField("sql", sqlStr).Debug()

	var args []interface{}
	for _, name := range names {
		args = append(args, name)
	}

	rows, err := cursor.Query(sqlStr, args...)
	if err != nil {
		return tagsIns, err
	}
	if closeRows {
		defer rows.Close()
	}

	for rows.Next() {
		var tagIns tagObj
		err := rows.Scan(&tagIns.ID, &tagIns.Name)
		if err != nil {
			return tagsIns, err
		}
		tagsIns = append(tagsIns, tagIns)
	}

	err = rows.Err()
	if err != nil {
		return tagsIns, err
	}

	return tagsIns, nil
}

func deleteUnusedTags(cursor cursorObj) (uint32, error) {
	sqlStr := `delete from notebook.tag
					where id in (
					  select * from (
					  select tag.id 
					  from notebook.tag tag
					  left outer join notebook.note_tag note_tag
					  on tag.id = note_tag.tag_id
					  where note_tag.id is null
					  ) as tmp
					)`
	result, err := cursor.Exec(sqlStr)
	if err != nil {
		return 0, err
	}
	deletedRows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint32(deletedRows), nil
}

func insertNote(cursor cursorObj, title string, author string, content string, plainText string, private bool, words uint32) (uint32, error) {
	sqlStr := `insert into notebook.note 
  				  	  (title, author, content, plain_text, words, private) 
  					  values 
  				  	  (?, ?, ?, ?, ?, ?)`
	result, err := cursor.Exec(sqlStr, title, author, content, plainText, words, private)
	if err != nil {
		return 0, err
	}

	noteID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint32(noteID), nil
}

func selectNoteByNoteID(cursor cursorObj, noteID uint32, closeRows bool) (*noteObj, error) {
	noteSqlIns := &noteSqlObj{}
	sqlStr := `select note.id, title, author, content, plain_text, words, private, 
					note.created_at, note.update_at, note_tag.tag_id, note_tag.tag_name
					from notebook.note
					left outer join notebook.note_tag note_tag
					on note.id = note_tag.note_id
					where note.id = ?`
	rows, err := cursor.Query(sqlStr, noteID)
	if err != nil {
		return nil, err
	}
	if closeRows {
		defer rows.Close()
	}

	var noteInsPtr *noteObj
	for rows.Next() {
		err := rows.Scan(&noteSqlIns.ID, &noteSqlIns.Title, &noteSqlIns.Author, &noteSqlIns.Content,
			&noteSqlIns.PlainText, &noteSqlIns.Words, &noteSqlIns.Auth, &noteSqlIns.CreatedAt, &noteSqlIns.UpdatedAt,
			&noteSqlIns.TagID, &noteSqlIns.TagName)
		if err != nil {
			return nil, err
		}

		if noteInsPtr == nil {
			noteInsPtr = &noteObj{
				ID: noteSqlIns.ID, Title: noteSqlIns.Title, Author: noteSqlIns.Author, Content: noteSqlIns.Content,
				PlainText: noteSqlIns.PlainText, Private: noteSqlIns.Auth, Words: noteSqlIns.Words,
				CreatedAt: noteSqlIns.CreatedAt, UpdateAt: noteSqlIns.UpdatedAt,
				Tags: []tagObj{}}
		}
		noteInsPtr.Tags = append(noteInsPtr.Tags, tagObj{ID: uint32(noteSqlIns.TagID.Int64), Name: noteSqlIns.TagName.String})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return noteInsPtr, nil
}

func selectNotesByTagID(cursor cursorObj, pageNo uint32, pageSize uint32, tagID uint32, autoSort bool, closeRows bool) ([]*noteObj, error) {
	// return pointer to local variable is ok in golang, see https://stackoverflow.com/questions/13715237/return-pointer-to-local-struct
	notesInsPtr := make([]*noteObj, 0)

	var sqlStr string
	var sqlSort string
	var rows *sql.Rows
	var err error

	if autoSort {
		sqlSort = "order by update_at desc"
	} else {
		sqlSort = ""
	}

	if tagID > 0 {
		sqlStr = `select note.id, title, author, content, plain_text, words, private, 
       			note.created_at, note.update_at,
					tag_id, tag_name from (
					select note.id, title, author, content, plain_text, words, private, 
       			note.created_at, note.update_at
					from notebook.note note
					inner join notebook.note_tag note_tag
					on note.id = note_tag.note_id
					and note_tag.tag_id = ?
					%s
					limit ?, ?) as note
					inner join notebook.note_tag note_tag
					on note.id = note_tag.note_id
					`
		sqlStr = fmt.Sprintf(sqlStr, sqlSort)
		rows, err = cursor.Query(sqlStr, tagID, (pageNo-1)*pageSize, pageSize)
	} else {
		sqlStr = `select note.id, title, author, content, plain_text, words, private, 
       				note.created_at, note.update_at,
						tag_id, tag_name from (
						select id, title, author, content, plain_text, words, private, 
       				created_at, update_at
						from notebook.note 
						%s
						limit ?, ?) as note
						left outer join notebook.note_tag note_tag
						on note.id = note_tag.note_id`
		sqlStr = fmt.Sprintf(sqlStr, sqlSort)
		rows, err = cursor.Query(sqlStr, (pageNo-1)*pageSize, pageSize)
	}

	if err != nil {
		return notesInsPtr, nil
	}

	if closeRows {
		defer rows.Close()
	}

	notesInsPtrMap := map[uint32]*noteObj{}
	for rows.Next() {
		var noteSqlIns noteSqlObj
		err := rows.Scan(&noteSqlIns.ID, &noteSqlIns.Title, &noteSqlIns.Author, &noteSqlIns.Content,
			&noteSqlIns.PlainText, &noteSqlIns.Words, &noteSqlIns.Auth,
			&noteSqlIns.CreatedAt, &noteSqlIns.UpdatedAt,
			&noteSqlIns.TagID, &noteSqlIns.TagName)
		if err != nil {
			return notesInsPtr, err
		}

		var noteInsPtr *noteObj
		if _, ok := notesInsPtrMap[noteSqlIns.ID]; !ok {
			noteInsPtr = &noteObj{
				ID: noteSqlIns.ID, Title: noteSqlIns.Title, Author: noteSqlIns.Author, Content: noteSqlIns.Content,
				PlainText: noteSqlIns.PlainText, Private: noteSqlIns.Auth, Words: noteSqlIns.Words,
				CreatedAt: noteSqlIns.CreatedAt, UpdateAt: noteSqlIns.UpdatedAt,
				Tags: []tagObj{}}
			notesInsPtrMap[noteSqlIns.ID] = noteInsPtr
		} else {
			noteInsPtr = notesInsPtrMap[noteSqlIns.ID]
		}

		if noteSqlIns.TagID.Valid {
			noteInsPtr.Tags = append(noteInsPtr.Tags, tagObj{ID: uint32(noteSqlIns.TagID.Int64), Name: noteSqlIns.TagName.String})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, v := range notesInsPtrMap {
		notesInsPtr = append(notesInsPtr, v)
	}

	if autoSort {
		sort.Sort(notesInsPtrSlice(notesInsPtr))
	}

	return notesInsPtr, nil
}

func selectNotesCountByTagID(cursor cursorObj, tagID uint32) (uint32, error) {
	var err error
	var sqlStr string
	var cnt uint32

	if tagID != 0 {
		sqlStr = `select count(distinct note.id)
						from notebook.note note
						inner join notebook.note_tag note_tag
						on note.id = note_tag.note_id
						and note_tag.tag_id = ?`
		err = cursor.QueryRow(sqlStr, tagID).Scan(&cnt)
	} else {
		sqlStr = "select count(id) from notebook.note"
		err = cursor.QueryRow(sqlStr).Scan(&cnt)
	}

	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func updateNoteByNoteID(cursor cursorObj, noteID uint32, title string, content string, plainText string, words uint32, private bool) (uint32, error) {
	sqlStr := `update notebook.note
					set title = ?, content = ?, plain_text = ?, words = ?, private = ?
					where id = ?`
	result, err := cursor.Exec(sqlStr, title, content, plainText, words, private, noteID)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint32(rowsAffected), nil
}

func deleteNoteByNoteID(cursor cursorObj, noteID uint32) (uint32, error) {
	sqlStr := "delete from notebook.note where id = ?"
	result, err := cursor.Exec(sqlStr, noteID)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint32(rowsAffected), nil
}

func insertNoteTags(cursor cursorObj, noteID uint32, tagsIns ...tagObj) (uint32, error) {
	var params []string
	for i := 0; i < len(tagsIns); i++ {
		params = append(params, fmt.Sprintf("(%d, ?, ?)", noteID))
	}
	sqlStr := fmt.Sprintf("insert into notebook.note_tag (note_id, tag_id, tag_name) values %s", strings.Join(params, ","))
	log.Logger.WithField("sql", sqlStr).Debug()

	var args []interface{}
	for _, tagIns := range tagsIns {
		args = append(args, tagIns.ID, tagIns.Name)
	}

	result, err := cursor.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return uint32(rowsAffected), nil
}

func deleteNoteTagsByNoteID(cursor cursorObj, noteID uint32) (uint32, error) {
	sqlStr := `delete from notebook.note_tag
					where note_id = ?`
	result, err := cursor.Exec(sqlStr, noteID)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return uint32(rowsAffected), nil
}
