package resource

import (
	"reflect"

	"regexp"

	"git.containerum.net/ch/grpc-proto-files/auth"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

type CreateResourceRequest struct {
	TariffID string `json:"tariff-id" binding:"uuid4"`
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

type SetResourcesAccessRequest struct {
	Access PermissionStatus `json:"access" binding:"eq=owner|eq=read|eq=write|eq=readdelete|eq=none"`
}

type ResizeResourceRequest struct {
	NewTariffID string `json:"tariff_id" binding:"uuid4"`
}

// Namespaces

type CreateNamespaceRequest = CreateResourceRequest

type GetUserNamespacesResponse []NamespaceWithVolumes

func (r GetUserNamespacesResponse) Mask() {
	for i := range r {
		r[i].Mask()
	}
}

type GetUserNamespaceResponse = NamespaceWithVolumes

type GetAllNamespacesResponse []NamespaceWithVolumes

func (r GetAllNamespacesResponse) Mask() {
	for i := range r {
		r[i].Mask()
	}
}

type GetUserNamespaceAccessesResponse = NamespaceWithUserPermissions

type RenameNamespaceRequest = RenameResourceRequest

type SetNamespaceAccessRequest struct {
	Username string           `json:"username" binding:"email"`
	Access   PermissionStatus `json:"access" binding:"eq=owner|eq=read|eq=write|eq=readdelete|eq=none"`
}

type ResizeNamespaceRequest = ResizeResourceRequest

// Volumes

type CreateVolumeRequest = CreateResourceRequest

type GetUserVolumesResponse []VolumeWithPermission

func (r GetUserVolumesResponse) Mask() {
	for i := range r {
		r[i].Mask()
	}
}

type GetUserVolumeResponse = VolumeWithPermission

type GetAllVolumesResponse []VolumeWithPermission

func (r GetAllVolumesResponse) Mask() {
	for i := range r {
		r[i].Mask()
	}
}

type GetVolumeAccessesResponse = VolumeWithUserPermissions

type RenameVolumeRequest = RenameResourceRequest

type SetVolumeAccessRequest = SetNamespaceAccessRequest

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

type AddDomainRequest = DomainEntry

type GetAllDomainsQueryParams struct {
	Page    int `form:"page" binding:"gt=0"`
	PerPage int `form:"per_page" binding:"gt=0"`
}

type GetAllDomainsResponse = []DomainEntry

type GetDomainResponse = DomainEntry

// Other

// GetUserAccessResponse is response for special request needed for auth server (actually for creating tokens)
type GetUserAccessesResponse = auth.ResourcesAccess

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

func RegisterCustomTagsGin(validate binding.StructValidator) error {
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

func dnsValidationFunc(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return dnsLabel.MatchString(field.String())
}

func dockerImageValidationFunc(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return dockerImage.MatchString(field.String())
}
