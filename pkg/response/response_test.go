package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponses(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		w := httptest.NewRecorder()
		OK(w, "Success item", map[string]string{"foo": "bar"})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var res APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.True(t, res.Success)
		assert.Equal(t, "Success item", res.Message)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("Created", func(t *testing.T) {
		w := httptest.NewRecorder()
		Created(w, "Item created", 123)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("NoContent", func(t *testing.T) {
		w := httptest.NewRecorder()
		NoContent(w)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("BadRequest", func(t *testing.T) {
		w := httptest.NewRecorder()
		BadRequest(w, "Invalid input", "field is missing")

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var res APIResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.False(t, res.Success)
		assert.Equal(t, "BAD_REQUEST", res.ErrorCode)
		assert.Equal(t, "field is missing", res.Errors)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		Unauthorized(w, "")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var res APIResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.Equal(t, "UNAUTHORIZED", res.ErrorCode)
		assert.Equal(t, "Authentication required or token expired", res.Message)
	})

	t.Run("Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		Forbidden(w, "Access Denied", nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var res APIResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.Equal(t, "FORBIDDEN", res.ErrorCode)
	})

	t.Run("NotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		NotFound(w, "User not found")

		assert.Equal(t, http.StatusNotFound, w.Code)
		var res APIResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		assert.Equal(t, "NOT_FOUND", res.ErrorCode)
	})

	t.Run("Conflict", func(t *testing.T) {
		w := httptest.NewRecorder()
		Conflict(w, "Email exists")

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("UnprocessableEntity", func(t *testing.T) {
		w := httptest.NewRecorder()
		UnprocessableEntity(w, "Validation error", map[string]string{"name": "required"})

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("TooManyRequests", func(t *testing.T) {
		w := httptest.NewRecorder()
		TooManyRequests(w, "Rate limit exceeded")

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("InternalServerError", func(t *testing.T) {
		w := httptest.NewRecorder()
		InternalServerError(w, "DB down", nil)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
