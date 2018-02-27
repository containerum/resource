package impl

import (
	"context"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateIngress(ctx context.Context, nsLabel string, req rstypes.CreateIngressRequest) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Infof("create ingress %#v", req)

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if createErr := tx.CreateIngress(ctx, userID, nsLabel, req); createErr != nil {
			return createErr
		}

		// TODO: create ingress in kube

		return nil
	})

	return err
}

func (rs *resourceServiceImpl) GetUserIngresses(ctx context.Context, nsLabel string,
	params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
		"user_id":  userID,
		"ns_label": nsLabel,
	}).Info("get user ingresses")

	resp, err := rs.DB.GetUserIngresses(ctx, userID, nsLabel, params)

	return resp, err
}

func (rs *resourceServiceImpl) GetAllIngresses(ctx context.Context, params rstypes.GetIngressesQueryParams) (rstypes.GetIngressesResponse, error) {
	rs.log.WithFields(logrus.Fields{
		"page":     params.Page,
		"per_page": params.PerPage,
	}).Info("get all ingresses")

	resp, err := rs.DB.GetAllIngresses(ctx, params)

	return resp, err
}

func (rs *resourceServiceImpl) DeleteIngress(ctx context.Context, nsLabel, domain string) error {
	userID := utils.MustGetUserID(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id":  userID,
		"ns_label": nsLabel,
		"domain":   domain,
	}).Info("delete ingress")

	err := rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if delErr := tx.DeleteIngress(ctx, userID, nsLabel, domain); delErr != nil {
			return delErr
		}

		// TODO: delete ingress in kube

		return nil
	})

	return err
}
