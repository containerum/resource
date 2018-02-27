package utils

import (
	"context"
	"net/http"
	"net/textproto"

	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/kube-client/pkg/cherry"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/gonic"
	"github.com/gin-gonic/gin"
)

var headersKey = new(struct{})

// SaveHeaders is a gin middleware which saves headers to request context
func SaveHeaders(ctx *gin.Context) {
	rctx := context.WithValue(ctx.Request.Context(), headersKey, ctx.Request.Header)
	ctx.Request = ctx.Request.WithContext(rctx)
}

// RequestHeadersMap extracts saved headers from context as map[string]string (useful for resty).
// saveHeaders middleware required for operation.
func RequestHeadersMap(ctx context.Context) map[string]string {
	ret := make(map[string]string)
	for k, v := range ctx.Value(headersKey).(http.Header) {
		if len(v) > 0 {
			ret[textproto.CanonicalMIMEHeaderKey(k)] = v[0] // this is how MIMEHeader.Get() works actually
		}
	}
	return ret
}

// RequestHeaders extracts saved headers from context.
// saveHeaders middleware required for operation.
func RequestHeaders(ctx context.Context) http.Header {
	return ctx.Value(headersKey).(http.Header)
}

var hdrToKey = map[string]interface{}{
	umtypes.UserIDHeader:      UserIDContextKey,
	umtypes.UserAgentHeader:   UserAgentContextKey,
	umtypes.FingerprintHeader: FingerPrintContextKey,
	umtypes.SessionIDHeader:   SessionIDContextKey,
	umtypes.TokenIDHeader:     TokenIDContextKey,
	umtypes.ClientIPHeader:    ClientIPContextKey,
	umtypes.UserRoleHeader:    UserRoleContextKey,
}

// RequireHeaders is a gin middleware to ensure that headers is set
func RequireHeaders(errToReturn cherry.Err, headers ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var notFoundHeaders []string
		for _, v := range headers {
			if ctx.GetHeader(v) == "" {
				notFoundHeaders = append(notFoundHeaders, v)
			}
		}
		if len(notFoundHeaders) > 0 {
			err := errToReturn.AddDetailF("required headers %v was not provided", notFoundHeaders)
			gonic.Gonic(err, ctx)
		}
	}
}

// PrepareContext is a gin middleware which adds values from header to context
func PrepareContext(ctx *gin.Context) {
	for hn, ck := range hdrToKey {
		if hv := ctx.GetHeader(hn); hv != "" {
			rctx := context.WithValue(ctx.Request.Context(), ck, hv)
			ctx.Request = ctx.Request.WithContext(rctx)
		}
	}
}

// RequireAdminRole is a gin middleware which requires admin role
func RequireAdminRole(errToReturn cherry.Err) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.GetHeader(umtypes.UserRoleHeader) != "admin" {
			gonic.Gonic(errToReturn.AddDetails("only admin can do this"), ctx)
		}
	}
}

// SubstituteUserMiddleware replaces user id in context with user id from query if it set and user is admin
func SubstituteUserMiddleware(ctx *gin.Context) {
	role := ctx.GetHeader(umtypes.UserRoleHeader)
	if userID, set := ctx.GetQuery("user-id"); set && role == "admin" {
		rctx := context.WithValue(ctx.Request.Context(), UserIDContextKey, userID)
		ctx.Request = ctx.Request.WithContext(rctx)
	}
}
