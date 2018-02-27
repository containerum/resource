package server

import (
	"sync"

	"git.containerum.net/ch/json-types/billing"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
)

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
		return rserrors.ErrTariffNotFound
	}
	if !isAdmin && !tariff.Public {
		return rserrors.ErrTariffNotFound
	}

	return nil
}

// VolumeLabelFromNamespaceLabel generates label for non-persistent volume
func VolumeLabelFromNamespaceLabel(label string) string {
	return label + "-volume"
}
