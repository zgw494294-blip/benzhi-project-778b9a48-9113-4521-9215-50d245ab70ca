package httpui

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"textilepermit/internal/domain"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Field   string `json:"field,omitempty"`
	} `json:"error"`
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("请求体只能包含一个 JSON 对象")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "服务暂时无法完成请求"
	field := ""
	var rule *domain.RuleError
	switch {
	case errors.As(err, &rule):
		status = http.StatusUnprocessableEntity
		code = "validation_error"
		message = rule.Message
		field = rule.Field
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = err.Error()
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		code = "version_conflict"
		message = err.Error()
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrSeparation):
		status = http.StatusConflict
		code = "state_conflict"
		message = err.Error()
	}
	var b errorBody
	b.Error.Code = code
	b.Error.Message = message
	b.Error.Field = field
	writeJSON(w, status, b)
}
func badJSON(w http.ResponseWriter, err error) {
	var b errorBody
	b.Error.Code = "invalid_json"
	b.Error.Message = "JSON 请求无效：" + err.Error()
	writeJSON(w, http.StatusBadRequest, b)
}
