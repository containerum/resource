package server

import (
	"git.containerum.net/ch/resource-service/pkg/models/service"
	"git.containerum.net/ch/resource-service/pkg/models/stats"
	"git.containerum.net/ch/resource-service/pkg/rsErrors"
	kubtypes "github.com/containerum/kube-client/pkg/model"
)

// DetermineServiceType deduces service type from service ports. If we have one or more "Port" set it is internal.
func DetermineServiceType(svc kubtypes.Service) service.ServiceType {
	serviceType := service.ServiceExternal
	for _, port := range svc.Ports {
		if port.Port != nil {
			serviceType = service.ServiceInternal
			break
		}
	}
	if svc.Domain != "" {
		serviceType = service.ServiceExternal
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

func CheckDeploymentCreateQuotas(ns kubtypes.Namespace, nsUsage kubtypes.Resource, deploy kubtypes.Deployment) error {
	CalculateDeployResources(&deploy)

	var deployCPU, deployRAM int
	deployCPU = int(deploy.TotalCPU)
	deployRAM = int(deploy.TotalCPU)

	if exceededCPU := int(ns.Resources.Hard.CPU) - deployCPU - int(nsUsage.CPU); exceededCPU < 0 {
		return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d CPU", -exceededCPU)
	}

	if exceededRAM := int(ns.Resources.Hard.Memory) - deployRAM - int(nsUsage.Memory); exceededRAM < 0 {
		return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d memory", -exceededRAM)
	}

	return nil
}

func CheckDeploymentReplaceQuotas(ns kubtypes.Namespace, nsUsage kubtypes.Resource, oldDeploy, newDeploy kubtypes.Deployment) error {
	CalculateDeployResources(&oldDeploy)

	var oldDeployCPU, oldDeployRAM int
	oldDeployCPU = int(oldDeploy.TotalCPU)
	oldDeployRAM = int(oldDeploy.TotalMemory)

	CalculateDeployResources(&newDeploy)

	var newDeployCPU, newDeployRAM int
	newDeployCPU = int(newDeploy.TotalCPU)
	newDeployRAM = int(newDeploy.TotalMemory)

	if exceededCPU := int(ns.Resources.Hard.CPU) - int(nsUsage.CPU) - newDeployCPU + oldDeployCPU; exceededCPU < 0 {
		return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d CPU", -exceededCPU)
	}

	if exceededRAM := int(ns.Resources.Hard.Memory) - int(nsUsage.Memory) - newDeployRAM + oldDeployRAM; exceededRAM < 0 {
		return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d memory", -exceededRAM)
	}

	return nil
}

func CheckDeploymentReplicasChangeQuotas(ns kubtypes.Namespace, nsUsage kubtypes.Resource, deploy kubtypes.Deployment, newReplicas int) error {
	CalculateDeployResources(&deploy)
	var deployCPU, deployRAM int
	deployCPU = int(deploy.TotalCPU)
	deployRAM = int(deploy.TotalMemory)

	if exceededCPU := int(ns.Resources.Hard.CPU) - int(nsUsage.CPU) - deployCPU*newReplicas + deployCPU*deploy.Replicas; exceededCPU < 0 {
		return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d CPU", -exceededCPU)
	}

	if exceededRAM := int(ns.Resources.Hard.CPU) - int(nsUsage.CPU) - deployRAM*newReplicas + deployRAM*deploy.Replicas; exceededRAM < 0 {
		return rserrors.ErrQuotaExceeded().AddDetailF("Exceeded %d memory", -exceededRAM)
	}

	return nil
}

func CheckServiceCreateQuotas(ns kubtypes.Namespace, nsUsage stats.Service, serviceType service.ServiceType) error {
	switch serviceType {
	case service.ServiceExternal:
		if int(ns.MaxExtService) <= nsUsage.External {
			return rserrors.ErrQuotaExceeded().AddDetailF("Maximum of external services reached")
		}
	case service.ServiceInternal:
		if int(ns.MaxIntService) <= nsUsage.Internal {
			return rserrors.ErrQuotaExceeded().AddDetailF("Maximum of internal services reached")
		}
	default:
		return rserrors.ErrValidation().AddDetailF("Invalid service type %s", serviceType)
	}
	return nil
}

func CalculateDeployResources(deploy *kubtypes.Deployment) {
	var mCPU, mbRAM int64
	for _, container := range deploy.Containers {
		mCPU += int64(container.Limits.CPU)
		mbRAM += int64(container.Limits.Memory)
	}
	mCPU *= int64(deploy.Replicas)
	mbRAM *= int64(deploy.Replicas)
	deploy.TotalCPU = uint(mCPU)
	deploy.TotalMemory = uint(mbRAM)
}
