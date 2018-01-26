package impl

import (
	"context"

	"strings"

	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/resource-service/models"
	"git.containerum.net/ch/resource-service/server"
	"git.containerum.net/ch/utils"
	"github.com/sirupsen/logrus"
)

func (rs *resourceServiceImpl) CreateNamespace(ctx context.Context, req *rstypes.CreateNamespaceRequest) (err error) {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"tariff_id": req.TariffID,
		"label":     req.Label,
		"user_id":   userID,
		"admin":     isAdmin,
	}).Infof("creating namespace for user")

	tariff, err := rs.Billing.GetNamespaceTariff(ctx, req.TariffID)
	if err != nil {
		return
	}

	if !isAdmin && !tariff.Public {
		err = server.ErrPermission
		return
	}

	newNamespace := rstypes.Namespace{
		Resource:            rstypes.Resource{TariffID: tariff.ID},
		RAM:                 tariff.MemoryLimit,
		CPU:                 tariff.CPULimit,
		MaxExternalServices: tariff.ExternalServices,
		MaxIntServices:      tariff.InternalServices,
		MaxTraffic:          tariff.Traffic,
	}

	err = rs.DB.Transactional(ctx, func(ctx context.Context, tx models.DB) error {
		if err := tx.CreateNamespace(ctx, userID, req.Label, &newNamespace); err != nil {
			return err
		}

		if err := rs.Kube.CreateNamespace(ctx,
			newNamespace.ID, newNamespace.CPU, newNamespace.RAM, req.Label, rstypes.PermissionStatusOwner); err != nil {
			return err
		}

		if err := rs.Billing.Subscribe(ctx, userID, newNamespace.Resource, rstypes.KindNamespace); err != nil {
			return err
		}

		// TODO: tariff activation

		// TODO: update user access

		return nil
	})
	if err = server.HandleDBError(err); err != nil {
		return
	}

	go func() {
		if err := rs.Mail.SendNamespaceCreated(ctx, userID, req.Label, tariff); err != nil {
			logrus.WithError(err).Error("send namespace created email failed")
		}
	}()

	return
}

func (rs *resourceServiceImpl) GetUserNamespaces(ctx context.Context,
	params *rstypes.GetAllResourcesQueryParams) (rstypes.GetAllNamespacesResponse, error) {
	userID := utils.MustGetUserID(ctx)
	isAdmin := server.IsAdminRole(ctx)
	rs.log.WithFields(logrus.Fields{
		"user_id": userID,
		"admin":   isAdmin,
	}).Infof("get user namespaces (params %#v)", params)

	filters := models.ParseNamespaceFilterParams(strings.Split(params.Filters, ",")...)
	ret, err := rs.DB.GetAllNamespaces(ctx, params.Page, params.PerPage, &filters)
	if err != nil {
		return nil, server.HandleDBError(err)
	}

	// remove some data for user
	if !isAdmin {
		for i := range ret {
			ret[i].AccessLevel = ret[i].NewAccessLevel
			ret[i].NewAccessLevel = ""

			ret[i].ID = ""
			ret[i].Limited = nil
			for j := range ret[i].Volume {
				ret[i].Volume[j].AccessLevel = ret[i].Volume[j].NewAccessLevel
				ret[i].Volume[j].NewAccessLevel = ""

				ret[i].Volume[j].ID = ""
				ret[i].Volume[j].Limited = nil
			}
		}
	}
	return ret, nil
}
