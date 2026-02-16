package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func (workerConfig *WorkerConfig) ProcessSession(w http.ResponseWriter, r *http.Request) {
	session := Session{}
	err := json.NewDecoder(r.Body).Decode(&session)
	if err != nil {
		log.Println("error decoding req body. err: ", err)
		RespondWithError(w, http.StatusBadRequest, "error decoding req body")
		return
	}
	if session.ID == uuid.Nil {
		log.Println("session id is required")
		RespondWithError(w, http.StatusBadRequest, "session id is required")
		return
	}
	if session.UserID == uuid.Nil {
		log.Println("user id is required")
		RespondWithError(w, http.StatusBadRequest, "user id is required")
		return
	}
	if session.JobTitle == "" {
		log.Println("job title is required")
		RespondWithError(w, http.StatusBadRequest, "job title is required")
		return
	}
	if session.JobDescription == "" {
		log.Println("job description is required")
		RespondWithError(w, http.StatusBadRequest, "job description is required")
		return
	}
	go processSession(session, *workerConfig)
	RespondWithJson(w, http.StatusOK, map[string]any{
		"message": "session processing started",
	})

}
