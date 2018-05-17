package middleware

import (
	"fmt"

	"net/textproto"

	"encoding/base64"

	"errors"

	"git.containerum.net/ch/resource-service/pkg/models/headers"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

type UserHeaderDataMap map[string]headers.UserHeaderData

const (
	UserNamespaces = "user-namespaces"
	UserVolumes    = "user-volumes"
	UserRole       = "user-role"
	UserID         = "user-id"

	RoleUser  = "user"
	RoleAdmin = "admin"
)

func RequiredUserHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logrus.WithField("Headers", ctx.Request.Header).Debug("Header list")
		notFoundHeaders := requireHeaders(ctx, httputil.UserRoleXHeader)
		if len(notFoundHeaders) > 0 {
			gonic.Gonic(rserrors.ErrRequiredHeadersNotProvided().AddDetails(notFoundHeaders...), ctx)
			return
		}
		/* Check User-Role and User-Namespace, X-User-Volume */
		if isUser, err := checkIsUserRole(GetHeader(ctx, httputil.UserRoleXHeader)); err != nil {
			logrus.WithField("Value", GetHeader(ctx, httputil.UserRoleXHeader)).WithError(err).Warn("check User-Role Error")
			gonic.Gonic(rserrors.ErrInvalidRole(), ctx)
		} else {
			//User-Role: user, check User-Namespace, X-User-Volume
			if isUser {
				notFoundHeaders := requireHeaders(ctx, httputil.UserRoleXHeader, httputil.UserNamespacesXHeader, httputil.UserVolumesXHeader, httputil.UserIDXHeader)
				if len(notFoundHeaders) > 0 {
					gonic.Gonic(rserrors.ErrRequiredHeadersNotProvided().AddDetails(notFoundHeaders...), ctx)
					return
				}
				userNs, errNs := checkUserNamespace(GetHeader(ctx, httputil.UserNamespacesXHeader))
				userVol, errVol := checkUserVolume(GetHeader(ctx, httputil.UserVolumesXHeader))
				if errNs != nil {
					logrus.WithField("Value", GetHeader(ctx, httputil.UserNamespacesXHeader)).WithError(errNs).Warn("Check User-Namespace header Error")
					gonic.Gonic(rserrors.ErrValidation().AddDetails(fmt.Sprintf("%v: %v", httputil.UserNamespacesXHeader, errNs)), ctx)
					return
				}
				if errVol != nil {
					logrus.WithField("Value", GetHeader(ctx, httputil.UserVolumesXHeader)).WithError(errVol).Warn("Check User-Volume header Error")
					gonic.Gonic(rserrors.ErrValidation().AddDetails(fmt.Sprintf("%v: %v", httputil.UserVolumesXHeader, errVol)), ctx)
					return
				}
				ctx.Set(UserNamespaces, userNs)
				ctx.Set(UserVolumes, userVol)
			}
		}
	}
}

func checkIsUserRole(userRole string) (bool, error) {
	switch userRole {
	case "", RoleAdmin:
		return false, nil
	case RoleUser:
		return true, nil
	}
	return false, errors.New("invalid user role")
}

func requireHeaders(ctx *gin.Context, headers ...string) (notFoundHeaders []string) {
	for _, v := range headers {
		if GetHeader(ctx, v) == "" {
			notFoundHeaders = append(notFoundHeaders, v)
		}
	}
	return
}

func GetHeader(ctx *gin.Context, header string) string {
	return ctx.GetHeader(textproto.CanonicalMIMEHeaderKey(header))
}

func checkUserNamespace(userNamespace string) (*UserHeaderDataMap, error) {
	return ParseUserHeaderData(userNamespace)
}

func checkUserVolume(userVolume string) (*UserHeaderDataMap, error) {
	return ParseUserHeaderData(userVolume)
}

//ParseUserHeaderData decodes headers for substitutions
func ParseUserHeaderData(str string) (*UserHeaderDataMap, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		logrus.WithError(err).WithField("Value", str).Warn("unable to decode user header data")
		return nil, errors.New("unable to decode user header data")
	}
	var userData []headers.UserHeaderData
	err = jsoniter.Unmarshal(data, &userData)
	if err != nil {
		logrus.WithError(err).WithField("Value", string(data)).Warn("unable to unmarshal user header data")
		return nil, errors.New("unable to unmarshal user header data")
	}
	result := UserHeaderDataMap{}
	for _, v := range userData {
		result[v.ID] = v
	}
	return &result, nil
}
