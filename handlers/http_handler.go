package handlers

import (
	"encoding/json"
	"net/http"
	"task-executor/define"

	"task-executor/executor"
	"task-executor/logger"
)

type HTTPHandler struct {
	executor *executor.Executor
}

func NewHTTPHandler(exec *executor.Executor) *HTTPHandler {
	return &HTTPHandler{
		executor: exec,
	}
}

// Beat handles heartbeat requests
func (h *HTTPHandler) Beat(w http.ResponseWriter, r *http.Request) {
	result := &define.ReturnT{
		Code: 200,
		Msg:  "success",
	}
	h.writeJSON(w, result)
}

// IdleBeat handles idle beat requests
func (h *HTTPHandler) IdleBeat(w http.ResponseWriter, r *http.Request) {
	var param define.IdleBeatParam
	if err := h.readJSON(r, &param); err != nil {
		h.writeErrorJSON(w, "Invalid request body")
		return
	}

	result := h.executor.IdleBeat(&param)
	h.writeJSON(w, result)
}

// Run handles job execution requests
func (h *HTTPHandler) Run(w http.ResponseWriter, r *http.Request) {
	var param define.TriggerParam
	if err := h.readJSON(r, &param); err != nil {
		h.writeErrorJSON(w, "Invalid request body")
		return
	}

	logger.Info("Received job execution request: JobID=%d, Handler=%s", param.JobID, param.ExecutorHandler)
	result := h.executor.ExecuteJob(&param)
	h.writeJSON(w, result)
}

// Kill handles job termination requests
func (h *HTTPHandler) Kill(w http.ResponseWriter, r *http.Request) {
	var param define.KillParam
	if err := h.readJSON(r, &param); err != nil {
		h.writeErrorJSON(w, "Invalid request body")
		return
	}

	logger.Info("Received job kill request: JobID=%d", param.JobID)
	result := h.executor.Kill(&param)
	h.writeJSON(w, result)
}

// Log handles log query requests
func (h *HTTPHandler) Log(w http.ResponseWriter, r *http.Request) {
	var param define.LogParam
	if err := h.readJSON(r, &param); err != nil {
		h.writeErrorJSON(w, "Invalid request body")
		return
	}

	result := h.executor.Log(&param)
	h.writeJSON(w, result)
}

func (h *HTTPHandler) readJSON(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (h *HTTPHandler) writeErrorJSON(w http.ResponseWriter, message string) {
	result := &define.ReturnT{
		Code: 500,
		Msg:  message,
	}
	h.writeJSON(w, result)
}
