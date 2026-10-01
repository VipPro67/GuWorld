package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
)

const redactedAuditValue = "[REDACTED]"

// AdminMutationAudit records one canonical event for an admin mutation,
// including failures. Request data is recursively redacted before persistence.
func AdminMutationAudit(action, targetType, permission string, destructive bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := readAndRestoreAuditBody(c)
		before, reason := sanitizedAuditChange(body, c)
		if reason == "" {
			reason = "API mutation: " + action
		}
		capture := &auditResponseWriter{ResponseWriter: c.Writer}
		c.Writer = capture

		c.Next()
		after := sanitizedAuditResponse(capture.body.Bytes())
		change := map[string]interface{}{"before": before, "after": after}

		actorID, hasActor := GetPlayerID(c)
		var actorIDPtr = &actorID
		if !hasActor {
			actorIDPtr = nil
		}
		username, _ := c.Get("username")
		traceID, _ := c.Get("trace_id")
		idempotencyKey, _ := c.Get("admin_idempotency_key")
		roles := []string{}
		legacyFallback := false
		if principal, ok := GetAdminPrincipal(c); ok {
			roles = append(roles, principal.Roles...)
			sort.Strings(roles)
			legacyFallback = principal.LegacyAdminFallback
		}
		status := c.Writer.Status()
		success := status >= http.StatusOK && status < http.StatusBadRequest
		errorMessage := ""
		if !success {
			errorMessage = fmt.Sprintf("HTTP %d %s", status, http.StatusText(status))
			monitor.GlobalAdminOperations.RecordFailedMutation(action)
			logger.L.Warn("admin_mutation_failed", zap.String("action", action), zap.String("permission", permission), zap.Int("status", status), zap.Any("trace_id", traceID), zap.String("player_id", actorID.String()))
			if strings.Contains(action, "import_") {
				monitor.GlobalAdminOperations.RecordImportFailure(action)
				logger.L.Error("admin_import_failed", zap.String("action", action), zap.Int("status", status), zap.Any("trace_id", traceID), zap.String("player_id", actorID.String()))
			}
		}
		if destructive {
			monitor.GlobalAdminOperations.RecordDestructiveAction(action)
			logger.L.Warn("admin_destructive_action", zap.String("action", action), zap.Bool("success", success), zap.Int("status", status), zap.Any("trace_id", traceID), zap.String("player_id", actorID.String()))
		}

		auditlog.Log(nil, auditlog.Entry{
			PlayerID:            actorIDPtr,
			ActorUsername:       fmt.Sprint(username),
			Action:              action,
			EffectiveRoles:      roles,
			EffectivePermission: permission,
			EntityType:          targetType,
			EntityID:            auditTargetID(c, before, after),
			ChangeData:          change,
			Reason:              reason,
			RequestID:           fmt.Sprint(traceID),
			IPAddress:           c.ClientIP(),
			Success:             success,
			ErrorMessage:        errorMessage,
			Metadata: map[string]interface{}{
				"route": c.FullPath(), "method": c.Request.Method,
				"legacy_admin_fallback": legacyFallback,
				"idempotency_key":       fmt.Sprint(idempotencyKey),
			},
		})
	}
}

func readAndRestoreAuditBody(c *gin.Context) []byte {
	if c.Request.Body == nil || strings.HasPrefix(c.ContentType(), "multipart/") {
		return nil
	}
	if c.Request.ContentLength > 1<<20 {
		return nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) > 1<<20 {
		// Never alter the downstream body, but avoid retaining oversized payloads.
		return nil
	}
	return body
}

func sanitizedAuditChange(body []byte, c *gin.Context) (interface{}, string) {
	change := map[string]interface{}{"path_params": auditPathParams(c)}
	if len(body) == 0 {
		return change, strings.TrimSpace(c.GetHeader("X-Admin-Reason"))
	}
	var payload interface{}
	if json.Unmarshal(body, &payload) != nil {
		change["request"] = "[NON_JSON_BODY]"
		return change, strings.TrimSpace(c.GetHeader("X-Admin-Reason"))
	}
	payload = redactAuditValue(payload)
	change["request"] = payload
	reason := strings.TrimSpace(c.GetHeader("X-Admin-Reason"))
	if object, ok := payload.(map[string]interface{}); ok {
		if value, ok := object["reason"].(string); ok && strings.TrimSpace(value) != "" {
			reason = strings.TrimSpace(value)
		}
	}
	return change, reason
}

func redactAuditValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			if isAuditSecretKey(key) {
				out[key] = redactedAuditValue
			} else {
				out[key] = redactAuditValue(child)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for index, child := range typed {
			out[index] = redactAuditValue(child)
		}
		return out
	default:
		return value
	}
}

func isAuditSecretKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(key))
	collapsed := strings.ReplaceAll(normalized, "_", "")
	for _, fragment := range []string{"password", "passwd", "token", "cookie", "otp", "authorization", "csrf", "paymentsecret", "clientsecret", "apisecret", "cardnumber", "cvv", "cvc"} {
		if strings.Contains(collapsed, fragment) {
			return true
		}
	}
	return normalized == "secret"
}

func auditPathParams(c *gin.Context) map[string]string {
	params := make(map[string]string, len(c.Params))
	for _, param := range c.Params {
		params[param.Key] = param.Value
	}
	return params
}

func auditTargetID(c *gin.Context, before, after interface{}) string {
	// The last path parameter is the most specific resource for nested routes.
	for index := len(c.Params) - 1; index >= 0; index-- {
		if value := c.Params[index].Value; value != "" {
			return value
		}
	}
	for _, value := range []interface{}{after, before} {
		if id := findAuditIdentifier(value); id != "" {
			return id
		}
	}
	return "collection"
}

func findAuditIdentifier(value interface{}) string {
	object, ok := value.(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"id", "key", "code", "instance_id", "pet_id"} {
		if candidate, exists := object[key]; exists && candidate != nil {
			return fmt.Sprint(candidate)
		}
	}
	for _, key := range []string{"data", "request"} {
		if candidate := findAuditIdentifier(object[key]); candidate != "" {
			return candidate
		}
	}
	return ""
}

func sanitizedAuditResponse(body []byte) interface{} {
	if len(body) == 0 {
		return nil
	}
	var payload interface{}
	if len(body) > 1<<20 || json.Unmarshal(body, &payload) != nil {
		return "[NON_JSON_OR_OVERSIZED_RESPONSE]"
	}
	return redactAuditValue(payload)
}

type auditResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (writer *auditResponseWriter) Write(data []byte) (int, error) {
	writer.body.Write(data)
	return writer.ResponseWriter.Write(data)
}

func (writer *auditResponseWriter) WriteString(data string) (int, error) {
	writer.body.WriteString(data)
	return writer.ResponseWriter.WriteString(data)
}
