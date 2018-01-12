package other

import (
	"fmt"

	"context"

	"git.containerum.net/ch/grpc-proto-files/auth"
	"git.containerum.net/ch/grpc-proto-files/common"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type AuthSvc interface {
	UpdateUserAccess(userID string) error
}

type authSvcGRPC struct {
	client auth.AuthClient
	addr   string
	log    *logrus.Entry
}

func NewAuthSvcGRPC(addr string) (as AuthSvc, err error) {
	ret := authSvcGRPC{
		log:  logrus.WithField("component", "auth_client"),
		addr: addr,
	}

	ret.log.Debugf("grpc connect to %s", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(ret.log)))
	if err != nil {
		return
	}
	ret.client = auth.NewAuthClient(conn)

	return ret, nil
}

func (as authSvcGRPC) UpdateUserAccess(userID string) error {
	as.log.WithField("user_id", userID).Infoln("update user access")
	_, err := as.client.UpdateAccess(context.Background(), &auth.UpdateAccessRequest{
		UserId: &common.UUID{Value: userID},
	})
	return err
}

func (as authSvcGRPC) String() string {
	return fmt.Sprintf("auth grpc client: addr=%v", as.addr)
}

type authSvcStub struct {
	log *logrus.Entry
}

func NewAuthSvcStub() AuthSvc {
	return authSvcStub{
		log: logrus.WithField("component", "auth_stub"),
	}
}

func (as authSvcStub) UpdateUserAccess(userID string) error {
	as.log.WithField("user_id", userID).Infoln("update user access")
	return nil
}

func (authSvcStub) String() string {
	return "ch-auth client dummy"
}
