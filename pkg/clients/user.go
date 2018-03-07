package clients

import (
	"context"

	"net/url"

	"git.containerum.net/ch/json-types/errors"
	umtypes "git.containerum.net/ch/json-types/user-manager"
	"git.containerum.net/ch/utils"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// UserManagerClient is interface to user-manager service
type UserManagerClient interface {
	UserInfoByLogin(ctx context.Context, login string) (*umtypes.UserInfoByLoginGetResponse, error)
	UserInfoByID(ctx context.Context, userID string) (*umtypes.UserInfoByIDGetResponse, error)
}

type httpUserManagerClient struct {
	log    *logrus.Entry
	client *resty.Client
}

// NewHTTPUserManagerClient returns rest-client to user-manager service
func NewHTTPUserManagerClient(url *url.URL) UserManagerClient {
	log := logrus.WithField("component", "user_manager_client")
	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(errors.Error{})
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &httpUserManagerClient{
		log:    log,
		client: client,
	}
}

func (u *httpUserManagerClient) UserInfoByLogin(ctx context.Context, login string) (*umtypes.UserInfoByLoginGetResponse, error) {
	u.log.WithField("login", login).Info("get user info by login")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(umtypes.UserInfoByLoginGetResponse{}).
		Get("/info/login/" + login)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err.(*errors.Error)
	}
	return resp.Result().(*umtypes.UserInfoByLoginGetResponse), nil
}

func (u *httpUserManagerClient) UserInfoByID(ctx context.Context, userID string) (*umtypes.UserInfoByIDGetResponse, error) {
	u.log.WithField("id", userID).Info("get user info by id")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(umtypes.UserInfoByLoginGetResponse{}).
		Get("/info/id/" + userID)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err.(*errors.Error)
	}
	return resp.Result().(*umtypes.UserInfoByIDGetResponse), nil
}

type userManagerStub struct {
	log         *logrus.Entry
	givenLogins map[string]umtypes.UserInfoByLoginGetResponse
}

func NewUserManagerStub() UserManagerClient {
	return &userManagerStub{
		log:         logrus.WithField("component", "user_manager_stub"),
		givenLogins: make(map[string]umtypes.UserInfoByLoginGetResponse),
	}
}

func (u *userManagerStub) UserInfoByLogin(ctx context.Context, login string) (*umtypes.UserInfoByLoginGetResponse, error) {
	u.log.WithField("id", login).Info("get user info by login")
	resp, ok := u.givenLogins[login]
	if !ok {
		resp = umtypes.UserInfoByLoginGetResponse{
			ID:   utils.NewUUID(),
			Role: "user",
			Data: map[string]interface{}{
				"email": login,
			},
		}
		u.givenLogins[login] = resp
	}
	return &resp, nil
}

func (u *userManagerStub) UserInfoByID(ctx context.Context, userID string) (*umtypes.UserInfoByIDGetResponse, error) {
	u.log.WithField("id", userID).Info("get user info by id")
	return &umtypes.UserInfoByIDGetResponse{
		Login: "fake-" + userID + "@test.com",
		Role:  "user",
		Data: map[string]interface{}{
			"email": "fake-" + userID + "@test.com",
		},
	}, nil
}
