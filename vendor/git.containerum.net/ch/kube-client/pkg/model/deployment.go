package model

type DeploymentStatus struct {
	CreatedAt           int64 `json:"created_at"`
	UpdatedAt           int64 `json:"updated_at"`
	Replicas            int   `json:"replicas"`
	ReadyReplicas       int   `json:"ready_replicas"`
	AvailableReplicas   int   `json:"available_replicas"`
	UnavailableReplicas int   `json:"unavailable_replicas"`
	UpdatedReplicas     int   `json:"updated_replicas"`
}

type UpdateReplicas struct {
	Replicas int `json:"replicas" binding:"required"`
}

type Deployment struct {
	Status     *DeploymentStatus `json:"status,omitempty"` //
	Containers []Container       `json:"containers"`       //
	Labels     map[string]string `json:"labels,omitempty"` //
	Name       string            `json:"name"`             // not UUID!
	Replicas   int               `json:"replicas"`         //
}

type Container struct {
	Image        string            `json:"image"`                   //
	Name         string            `json:"name"`                    // not UUID!
	Limits       Resource          `json:"limits"`                  //
	Env          []Env             `json:"env,omitempty"`           //
	Commands     []string          `json:"commands,omitempty"`      //
	Ports        []ContainerPort   `json:"ports,omitempty"`         //
	VolumeMounts []ContainerVolume `json:"volume_mounts,omitempty"` //
	ConfigMaps   []ContainerVolume `json:"config_maps,omitempty"`   //
}

type Env struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type ContainerPort struct {
	Name     string   `json:"name"`
	Port     int      `json:"port"`
	Protocol Protocol `json:"protocol"`
}

type ContainerVolume struct {
	Name      string  `json:"name"`
	Mode      *string `json:"mode,omitempty"`
	MountPath string  `json:"mount_path"`
	SubPath   *string `json:"sub_path,omitempty"`
}
