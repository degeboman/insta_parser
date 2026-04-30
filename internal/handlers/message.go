package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type telegramSender interface {
	SendMessage(text string) error
}

type MessageHandler struct {
	tg telegramSender
}

func NewMessageHandler(tg telegramSender) *MessageHandler {
	return &MessageHandler{tg: tg}
}

type sendRequest struct {
	Name      string `json:"name"`
	Attending string `json:"attending"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Send handles POST /send
// Body: {"message": "your text here"}
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{"method not allowed"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"failed to read body"})
		return
	}
	defer r.Body.Close()

	var req sendRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{"invalid JSON"})
		return
	}

	var text string

	if req.Attending == "yes" {
		text = fmt.Sprintf("Новый гость подтвердил участие!\n%s", req.Name)
	} else {
		text = fmt.Sprintf("Гость не придет((( \nЭто %s", req.Name)
	}

	if err := h.tg.SendMessage(text); err != nil {
		log.Printf("telegram send error: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to send message"})
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
