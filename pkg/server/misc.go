package server

import (
	"sync"

	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"git.containerum.net/ch/json-types/billing"
	rstypes "git.containerum.net/ch/json-types/resource-service"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	kubtypes "git.containerum.net/ch/kube-client/pkg/model"
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
		return rserrors.ErrTariffNotFound()
	}
	if !isAdmin && !tariff.Public {
		return rserrors.ErrTariffNotFound()
	}

	return nil
}

// VolumeLabel generates label for non-persistent volume
func VolumeLabel(nsLabel string) string {
	return nsLabel + "-volume"
}

// IngressName generates ingress name
func IngressName(domain string) string {
	return domain + "-ingress"
}

// SecretName generates secret name for ingress
func SecretName(ingressName string) string {
	return ingressName + "-secret"
}

// DetermineServiceType deduces service type from service ports. If we have one or more "TargetPort" set it is internal.
func DetermineServiceType(service kubtypes.Service) rstypes.ServiceType {
	serviceType := rstypes.ServiceExternal
	for _, port := range service.Ports {
		if port.TargetPort != nil {
			serviceType = rstypes.ServiceInternal
			break
		}
	}
	return serviceType
}

// IngressPaths generates ingress paths by service ports
func IngressPaths(service kubtypes.Service, path string, servicePort int) ([]kubtypes.Path, error) {
	if DetermineServiceType(service) != rstypes.ServiceExternal {
		return nil, rserrors.ErrServiceNotExternal()
	}

	var portExist bool
	for _, port := range service.Ports {
		if port.Port == servicePort && port.Protocol == kubtypes.TCP {
			portExist = true
			break
		}
	}
	if !portExist {
		return nil, rserrors.ErrTCPPortNotFound().AddDetailF("TCP port %d not exists in service %s", servicePort, service.Name)
	}

	ret := []kubtypes.Path{
		{Path: path, ServiceName: service.Name, ServicePort: servicePort},
	}

	return ret, nil
}

// VolumeGlusterName generates volume name for glusterfs (non-persistent volumes)
func VolumeGlusterName(nsLabel, userID string) string {
	glusterName := sha256.Sum256([]byte(fmt.Sprintf("%s-volume%s", nsLabel, userID)))
	return hex.EncodeToString(glusterName[:])
}
