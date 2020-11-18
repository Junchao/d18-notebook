package server

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/PuerkitoBio/goquery"
	"github.com/speed18/d18-notebook/log"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

// ------------------------------------------------------------------

func hideNote(noteInsPtr *noteObj) {
	noteInsPtr.Content = ""
	noteInsPtr.PlainText = ""
}

func cutNote(noteInsPtr *noteObj) {
	digestLength := viper.GetUint32("note.digest_length")
	// copy is needed so that the original plain text can be garbage collected
	part := make([]rune, digestLength)
	copy(part, []rune(noteInsPtr.PlainText)[:min(int(noteInsPtr.Words), int(digestLength))])
	noteInsPtr.PlainText = string(part)
	noteInsPtr.Content = ""
}

func extractPlainTextFromHTML(content string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", err
	}
	plainText := doc.Text()
	plainText = strings.Join(strings.Fields(plainText), "")
	log.Logger.WithField("plainText", plainText).Debug("")
	return plainText, nil
}

func calcPage(curPage int, pageSize int, totalCount int) pageObj {
	if pageSize <= 0 || curPage <= 0 || totalCount <= 0 {
		return pageObj{Left: 0, Right: 0, Cur: 0, Total: 0}
	}

	totalPage := (totalCount + pageSize - 1) / pageSize
	if curPage > totalPage {
		return pageObj{Left: 0, Right: 0, Cur: 0, Total: 0}
	}

	winSize := viper.GetInt("pagination.win_size")
	dist := winSize / 2
	left := curPage - dist
	right := curPage + dist

	if left <= 0 {
		left = 1
		right = min(winSize, totalPage)
	} else if right > totalPage {
		right = totalPage
		left = max(totalPage-winSize+1, 1)
	}

	return pageObj{Left: uint32(left), Right: uint32(right), Cur: uint32(curPage), Total: uint32(totalPage)}
}

func getNotes(pageNo uint32, tagID uint32, token string) ([]*noteObj, pageObj, error) {
	var pageIns pageObj
	notesInsPtr := make([]*noteObj, 0)

	notesCount, err := selectNotesCountByTagID(DB, tagID)
	if err != nil {
		return notesInsPtr, pageIns, err
	}
	pageSize := viper.GetUint32("pagination.page_size")
	pageIns = calcPage(int(pageNo), int(pageSize), int(notesCount))
	log.Logger.WithField("calc page result", pageIns).Debug()
	if pageIns.Total <= 0 {
		return notesInsPtr, pageIns, nil
	}

	notesInsPtr, err = selectNotesByTagID(DB, pageIns.Cur, pageSize, tagID, true, true)
	if err != nil {
		return notesInsPtr, pageIns, err
	}

	_isAuth := isAuth(token)
	for _, noteInsPtr := range notesInsPtr {
		if noteInsPtr.Private && !_isAuth {
			hideNote(noteInsPtr)
		} else {
			cutNote(noteInsPtr)
		}
	}

	return notesInsPtr, pageIns, err
}

func getNote(noteID uint32, token string) (*noteObj, error) {
	noteInsPtr, err := selectNoteByNoteID(DB, noteID, true)
	if err != nil {
		return noteInsPtr, err
	}
	if noteInsPtr == nil {
		return nil, noteNotExistsErr
	}
	if noteInsPtr.Private && !isAuth(token) {
		hideNote(noteInsPtr)
	}
	return noteInsPtr, nil
}

func publishNote(title string, content string, private bool, tagsName ...string) (uint32, error) {
	ret, err := withTransaction(func(cursor cursorObj) (i interface{}, e error) {
		var words uint32
		plainText, err := extractPlainTextFromHTML(content)
		if err != nil {
			log.Logger.WithField("err", err).Warn("extract plain text failed")
			words = 0
		} else {
			words = uint32(len([]rune(plainText)))
		}

		author := viper.GetString("note.default_author")
		noteID, err := insertNote(cursor, title, author, content, plainText, private, uint32(words))
		if err != nil {
			return 0, err
		}

		_, err = insertTags(cursor, true, tagsName...)
		if err != nil {
			return 0, err
		}

		// lock is required in case tags are deleted by "deleteUnusedTags"
		tagsIns, err := selectTagsByName(cursor, false, true, tagsName...)
		if err != nil {
			return 0, err
		}

		_, err = insertNoteTags(cursor, noteID, tagsIns...)
		if err != nil {
			return 0, err
		}

		return noteID, nil
	})

	if err != nil {
		return 0, err
	}

	noteID := ret.(uint32)
	return noteID, nil
}

func updateNote(noteID uint32, title string, content string, private bool, tagsName ...string) error {
	_, err := withTransaction(func(cursor cursorObj) (i interface{}, e error) {
		var words uint32
		plainText, err := extractPlainTextFromHTML(content)
		if err != nil {
			log.Logger.WithField("err", err).Warn("extract plain text failed")
			words = 0
		} else {
			words = uint32(len([]rune(plainText)))
		}

		rowsAffected, err := updateNoteByNoteID(cursor, noteID, title, content, plainText, words, private)
		if err != nil {
			return 0, err
		}
		//if rowsAffected <= 0 {
		//	return 0, noteNotExistsErr
		//}

		_, err = deleteNoteTagsByNoteID(cursor, noteID)
		if err != nil {
			return 0, err
		}

		_, err = insertTags(cursor, true, tagsName...)
		if err != nil {
			return 0, err
		}

		// lock is required in case tags are deleted by func "deleteUnusedTags"
		tagsIns, err := selectTagsByName(cursor, false, true, tagsName...)
		if err != nil {
			return 0, err
		}

		_, err = insertNoteTags(cursor, noteID, tagsIns...)
		if err != nil {
			return 0, err
		}

		return rowsAffected, nil
	})

	if err != nil {
		return err
	}

	go cleanUnusedTags()

	return nil
}

func deleteNote(noteID uint32) error {
	_, err := withTransaction(func(cursor cursorObj) (i interface{}, e error) {
		rowsAffected, err := deleteNoteByNoteID(cursor, noteID)
		if err != nil {
			return 0, err
		}
		if rowsAffected <= 0 {
			return 0, noteNotExistsErr
		}

		_, err = deleteNoteTagsByNoteID(cursor, noteID)
		if err != nil {
			return 0, err
		}

		return rowsAffected, nil
	})

	if err != nil {
		return err
	}

	go cleanUnusedTags()

	return nil
}

func getTags() ([]tagObj, error) {
	tagsIns, err := selectTagsWithNotesCount(DB, true)
	if err != nil {
		return tagsIns, err
	}
	return tagsIns, nil
}

func cleanUnusedTags() {
	log.Logger.Info("begin to clean up unused tags...")
	if deletedTagsNum, err := deleteUnusedTags(DB); err != nil {
		log.Logger.WithField("err", err).Warn("clean up unused tags error")
	} else {
		log.Logger.WithField("num of unused tags deleted", deletedTagsNum).Info()
	}
	log.Logger.Info("done cleaning up unused tags")
}

func genToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	tokenStr := base64.StdEncoding.EncodeToString(token)
	return tokenStr, nil
}

func getToken() (string, error) {
	token, err := RDS.Get(tokenKey).Result()
	if err != nil {
		return "", err
	}
	return token, err
}

func setToken(val string) error {
	if err := RDS.Set(tokenKey, val, time.Second*time.Duration(tokenExpire)).Err(); err != nil {
		return err
	}
	return nil
}

func deleteToken() error {
	if err := RDS.Del(tokenKey).Err(); err != nil {
		return err
	}
	return nil
}

func auth(password string) (string, error) {
	serverPassword := []byte(viper.GetString("auth.hashed_password"))
	if err := bcrypt.CompareHashAndPassword(serverPassword, []byte(password)); err != nil {
		return "", err
	}

	token, err := genToken()
	if err != nil {
		return "", err
	}
	log.Logger.WithField("gen token", token).Debug()

	if err := setToken(token); err != nil {
		return "", err
	}

	return token, nil
}

func isAuth(token string) bool {
	if token == "" {
		return false
	} else if serverToken, err := getToken(); err != nil {
		log.Logger.WithField("err", err).Warn("get token failed")
		return false
	} else {
		return serverToken == token
	}
}
