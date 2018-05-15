package server

import (
	"sync"

	rstypes "git.containerum.net/ch/resource-service/pkg/model"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	kubtypes "github.com/containerum/kube-client/pkg/model"
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

// VolumeLabel generates label for non-persistent volume
func VolumeLabel(nsLabel string) string {
	return nsLabel + "-volume"
}

// DetermineServiceType deduces service type from service ports. If we have one or more "Port" set it is internal.
func DetermineServiceType(service kubtypes.Service) rstypes.ServiceType {
	serviceType := rstypes.ServiceExternal
	for _, port := range service.Ports {
		if port.Port != nil {
			serviceType = rstypes.ServiceInternal
			break
		}
	}
	return serviceType
}

// IngressPaths generates ingress paths by service ports
func IngressPaths(service kubtypes.Service, path string, servicePort int) ([]kubtypes.Path, error) {
	var portExist bool
	for _, port := range service.Ports {
		if port.Port != nil && *port.Port == servicePort && port.Protocol == kubtypes.TCP {
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

/*
func GetAndCheckPermission(ctx context.Context, userID string, resourceKind rstypes.Kind, resourceName string, needed rstypes.PermissionStatus) error {
	if IsAdminRole(ctx) {
		return nil
	}

	/*	current, err := db.GetUserResourceAccess(ctx, userID, resourceKind, resourceName)
		if err != nil {
			return err
		}

		if !models.PermCheck(current, needed) {
			return rserrors.ErrPermissionDenied().AddDetailF("permission '%s' required for operation, you have '%s'", needed, current)
		}

	return nil
}
*/
/*
func CheckDeploymentCreateQuotas(ns rstypes.Namespace, nsUsage models.NamespaceUsage, deploy kubtypes.Deployment) error {
	if err := CalculateDeployResources(&deploy); err != nil {
		return err
	}

	var deployCPU, deployRAM int
	deployCPU = int(deploy.TotalCPU)
	deployRAM = int(deploy.TotalCPU)

	if exceededCPU := ns.CPU - deployCPU - nsUsage.CPU; exceededCPU < 0 {
		//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d CPU", -exceededCPU)
	}

	if exceededRAM := ns.RAM - deployRAM - nsUsage.RAM; exceededRAM < 0 {
		//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d memory", -exceededRAM)
	}

	return nil
}

func CheckDeploymentReplaceQuotas(ns rstypes.Namespace, nsUsage models.NamespaceUsage, oldDeploy, newDeploy kubtypes.Deployment) error {
	if err := CalculateDeployResources(&oldDeploy); err != nil {
		return err
	}

	var oldDeployCPU, oldDeployRAM int
	oldDeployCPU = int(oldDeploy.TotalCPU)
	oldDeployRAM = int(oldDeploy.TotalMemory)

	if err := CalculateDeployResources(&newDeploy); err != nil {
		return err
	}

	var newDeployCPU, newDeployRAM int
	newDeployCPU = int(newDeploy.TotalCPU)
	newDeployRAM = int(newDeploy.TotalMemory)

	if exceededCPU := ns.CPU - nsUsage.CPU - newDeployCPU + oldDeployCPU; exceededCPU < 0 {
		//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d CPU", -exceededCPU)
	}

	if exceededRAM := ns.CPU - nsUsage.CPU - newDeployRAM + oldDeployRAM; exceededRAM < 0 {
		//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d memory", -exceededRAM)
	}

	return nil
}


func CheckDeploymentReplicasChangeQuotas(ns rstypes.Namespace, nsUsage models.NamespaceUsage, deploy kubtypes.Deployment, newReplicas int) error {
	if err := CalculateDeployResources(&deploy); err != nil {
		return err
	}

	var deployCPU, deployRAM int
	deployCPU = int(deploy.TotalCPU)
	deployRAM = int(deploy.TotalMemory)

	if exceededCPU := ns.CPU - nsUsage.CPU - deployCPU*newReplicas + deployCPU*deploy.Replicas; exceededCPU < 0 {
		//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d CPU", -exceededCPU)
	}

	if exceededRAM := ns.CPU - nsUsage.CPU - deployRAM*newReplicas + deployRAM*deploy.Replicas; exceededRAM < 0 {
		//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d memory", -exceededRAM)
	}

	return nil
}
*/
/*
func CheckServiceCreateQuotas(ns rstypes.Namespace, nsUsage models.NamespaceUsage, serviceType rstypes.ServiceType) error {
	switch serviceType {
	case rstypes.ServiceExternal:
		if ns.MaxExternalServices <= nsUsage.ExtServices {
			//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Maximum of external services reached")
		}
	case rstypes.ServiceInternal:
		if ns.MaxIntServices <= nsUsage.IntServices {
			//TODO	return rserrors.ErrQuotaExceeded().AddDetailF("Maximum of internal services reached")
		}
	default:
		return rserrors.ErrValidation().AddDetailF("Invalid service type %s", serviceType)
	}
	return nil
}
*/
func CalculateDeployResources(deploy *kubtypes.Deployment) error {
	var mCPU, mbRAM int64
	for _, container := range deploy.Containers {
		mCPU += int64(container.Limits.CPU)
		mbRAM += int64(container.Limits.Memory)
	}
	mCPU *= int64(deploy.Replicas)
	mbRAM *= int64(deploy.Replicas)
	deploy.TotalCPU = uint(mCPU)
	deploy.TotalMemory = uint(mbRAM)
	return nil
}
