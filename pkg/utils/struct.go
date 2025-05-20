package utils

type Policy struct {
	Network []string
}

type Models struct {
	Network string
}

type Configuration struct {
	User struct {
		HuggingfaceToken string `yaml:"huggingface-token"`
		Home             string `yaml:"home"`
	} `yaml:"user"`
}

type Prompt struct {
	Network []string
}

type Entity struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type SamplePodInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type PodCommunication struct {
	SrcPodInfo NetInfo
	DstPodInfo NetInfo
	Related    bool
}

type NetInfo struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Labels         map[string]string `json:"labels"`
	ContainerPorts string            `json:"containerPorts"`
	Protocol       string            `json:"protocol"`
	ServiceName    string            `json:"serviceName"`
	RelatedEnv     []Service         `json:"relatedEnv"`
}

type Service struct {
	Name string
	Port string
}

type PodTraffic struct {
	SrcPod    string
	DstPod    string
	Direction string
	Protocol  string
	Port      int
	Status    string
}

type YAMLInfo struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}
