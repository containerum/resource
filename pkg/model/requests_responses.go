package model

import (
	"regexp"

	"time"

	"git.containerum.net/ch/auth/proto"
	"gopkg.in/go-playground/validator.v9"
)

type CreateResourceRequest struct {
	TariffID string `json:"tariff_id" binding:"uuid"`
	Label    string `json:"label" binding:"required,dns"`
}

type RenameResourceRequest struct {
	NewLabel string `json:"label" binding:"required,dns"`
}

type GetAllResourcesQueryParams struct {
	Page    int    `form:"page" binding:"gt=0"`
	PerPage int    `form:"per_page" binding:"gt=0"`
	Filters string `form:"filters"`
}

type ResizeResourceRequest struct {
	NewTariffID string `json:"tariff_id" binding:"uuid"`
}

type DeleteResourceAccessRequest struct {
	Username string `json:"username" binding:"email"`
}

// Namespaces

type CreateNamespaceRequest = CreateResourceRequest

type RenameNamespaceRequest = RenameResourceRequest

type DeleteNamespaceAccessRequest = DeleteResourceAccessRequest

type ResizeNamespaceRequest = ResizeResourceRequest

// Volumes

type CreateVolumeRequest = CreateResourceRequest

type RenameVolumeRequest = RenameResourceRequest

type DeleteVolumeAccessRequest = DeleteResourceAccessRequest

type ResizeVolumeRequest = ResizeResourceRequest

// Deployments

type SetReplicasRequest struct {
	Replicas int `json:"replicas" binding:"gte=1,lte=15"`
}

type SetContainerImageRequest struct {
	ContainerName string `json:"container_name" binding:"required,dns"`
	Image         string `json:"image" binding:"required,docker_image"`
}

// Domains

type AddDomainRequest = Domain

type GetAllDomainsQueryParams struct {
	Page    int `form:"page" binding:"gt=0"`
	PerPage int `form:"per_page" binding:"gt=0"`
}

type GetAllDomainsResponse = []Domain

type GetDomainResponse = Domain

// Ingresses

// Ingress is a basic type for ingress-related responses
type Ingress struct {
	Domain      string      `json:"domain" binding:"required"`
	Type        IngressType `json:"type" binding:"eq=http|eq=https|eq=custom_https"`
	Service     string      `json:"service" binding:"required,dns"`
	CreatedAt   *time.Time  `json:"created_at,omitempty" binding:"-"`
	Path        string      `json:"path"`
	ServicePort int         `json:"service_port" binding:"min=1,max=65535"`
}

type IngressTLS struct {
	Cert string `json:"crt" binding:"base64"`
	Key  string `json:"key" binding:"base64"`
}

type CreateIngressRequest struct {
	Ingress
	TLS *IngressTLS `json:"tls,omitempty" binding:"omitempty"`
}

type GetIngressesResponse []Ingress

type GetIngressesQueryParams struct {
	Page    int `form:"page" binding:"gt=0"`
	PerPage int `form:"per_page" binding:"gt=0"`
}

type UpdateIngressRequest struct {
	Service string `json:"service" binding:"required,dns"`
}

// Storages

type CreateStorageRequest struct {
	Name     string   `json:"name" binding:"required"`
	Size     int      `json:"size" binding:"gt=0"`
	Replicas int      `json:"replicas" binding:"gt=0"`
	IPs      []string `json:"ips" binding:"gt=0"`
}

type GetStoragesResponse []Storage

type UpdateStorageRequest struct {
	Name     *string  `json:"name,omitempty"`
	Size     *int     `json:"size,omitempty" binding:"omitempty,gt=0"`
	Replicas *int     `json:"replicas" binding:"omitempty,gt=0"`
	IPs      []string `json:"ips" binding:"omitempty,gt=0"`
}

// Other

type GetResourcesCountResponse struct {
	Namespaces  int `json:"namespaces"`
	Volumes     int `json:"volumes"`
	Deployments int `json:"deployments"`
	ExtServices int `json:"external_services"`
	IntServices int `json:"internal_services"`
	Ingresses   int `json:"ingresses"`
	Pods        int `json:"pods"`
	Containers  int `json:"containers"`
}

// GetUserAccessResponse is response for special request needed for auth server (actually for creating tokens)
type GetUserAccessesResponse = authProto.ResourcesAccess

// custom tag registration

var funcs = map[string]validator.Func{
	"dns":          dnsValidationFunc,
	"docker_image": dockerImageValidationFunc,
}

func RegisterCustomTags(validate *validator.Validate) error {
	for tag, fn := range funcs {
		if err := validate.RegisterValidation(tag, fn); err != nil {
			return err
		}
	}
	return nil
}

var (
	dnsLabel    = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	dockerImage = regexp.MustCompile(`(?:.+/)?([^:]+)(?::.+)?`)
)

func dnsValidationFunc(fl validator.FieldLevel) bool {
	return dnsLabel.MatchString(fl.Field().String())
}

func dockerImageValidationFunc(fl validator.FieldLevel) bool {
	return dockerImage.MatchString(fl.Field().String())
}
