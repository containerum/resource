package server

import (
	"net/http"

	"sync"

	"git.containerum.net/ch/json-types/billing"
	"git.containerum.net/ch/json-types/errors"
	"git.containerum.net/ch/resource-service/models"
)

// HandleDBError translates database errors to one type *"git.containerum.net/ch/json-types/errors".Error
func HandleDBError(err error) error {
	switch err {
	case nil:
		return nil
	case models.ErrTransactionRollback, models.ErrTransactionBegin, models.ErrTransactionCommit:
		return errors.NewWithCode(err.Error(), http.StatusInternalServerError)
	case models.ErrLabeledResourceNotExists, models.ErrResourceNotExists:
		return errors.NewWithCode(err.Error(), http.StatusNotFound)
	case models.ErrLabeledResourceExists, models.ErrResourceExists, models.ErrIngressExists:
		return errors.NewWithCode(err.Error(), http.StatusConflict)
	}

	switch err.(type) {
	case *models.DBError:
		return errors.NewWithCode(err.Error(), http.StatusInternalServerError)
	case *errors.Error:
		return err
	default:
		return errors.NewWithCode(err.Error(), http.StatusInternalServerError)
	}
}

// Parallel runs functions in dedicated goroutines and waits for ending
func Parallel(funcs ...func() error) (ret []error) {
	wg := &sync.WaitGroup{}
	wg.Add(len(funcs))
	retmu := &sync.Mutex{}
	for _, f := range funcs {
		go func(inf func() error) {
			if err := inf(); err != nil {
				retmu.Lock()
				defer retmu.Unlock()
				ret = append(ret, err)
			}
			wg.Done()
		}(f)
	}
	wg.Wait()
	return
}

// CheckTariff checks if user has permissions to use tariff
func CheckTariff(tariff billing.Tariff, isAdmin bool) error {
	if !tariff.Active {
		return ErrTariffInactive
	}
	if !isAdmin && !tariff.Public {
		return ErrTariffNotPublic
	}

	return nil
}

// VolumeLabelFromNamespaceLabel generates label for non-persistent volume
func VolumeLabelFromNamespaceLabel(label string) string {
	return label + "-volume"
}
