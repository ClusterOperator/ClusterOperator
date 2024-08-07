package constant

const (
	ClusterNotConnected = "NotConnected"
	ClusterTerminated   = "Terminated"

	NodeNameRuleDefault  = "default"
	NodeNameRuleHostName = "hostname"
	NodeNameRuleIP       = "ip"

	ClusterSourceLocal      = "local"
	ClusterNotReady         = "NotReady"
	ClusterSourceInternal   = "internal"
	ClusterSourceExternal   = "external"
	ClusterSourceKoExternal = "ko-external"

	NodeRoleNameMaster = "master"
	NodeRoleNameWorker = "worker"
	LbModeInternal     = "internal"

	ClusterProviderBareMetal = "bareMetal"
	ClusterProviderPlan      = "plan"

	AuthenticationModeBearer      = "bearer"
	AuthenticationModeCertificate = "certificate"
	AuthenticationModeConfigFile  = "configFile"

	DefaultNamespace     = "kube-operator"
	F5Namespace          = "kube-system"
	DefaultApiServerPort = 8443

	DefaultIngress            = "apps.ko.com"
	DefaultPrometheusIngress  = "prometheus." + DefaultIngress
	DefaultLoggingIngress     = "logging." + DefaultIngress
	DefaultLokiIngress        = "loki." + DefaultIngress
	DefaultGrafanaIngress     = "grafana." + DefaultIngress
	DefaultChartmuseumIngress = "chartmuseum." + DefaultIngress
	DefaultRegistryIngress    = "registry." + DefaultIngress
	DefaultDashboardIngress   = "dashboard." + DefaultIngress
	DefaultGatekeeperIngress  = "gatekeeper." + DefaultIngress
	DefaultKubeappsIngress    = "kubeapps." + DefaultIngress

	ChartmuseumChartName    = "nexus/chartmuseum"
	DockerRegistryChartName = "nexus/docker-registry"
	PrometheusChartName     = "nexus/prometheus"
	LoggingChartName        = "nexus/logging"
	LokiChartName           = "nexus/loki-stack"
	GrafanaChartName        = "nexus/grafana"
	DashboardChartName      = "nexus/kubernetes-dashboard"
	GatekeeperChartName     = "nexus/gatekeeper"
	KubeappsChartName       = "nexus/kubeapps"

	DefaultRegistryServiceName    = "registry-docker-registry"
	DefaultChartmuseumServiceName = "chartmuseum-chartmuseum"
	DefaultDashboardServiceName   = "dashboard-kubernetes-dashboard"
	DefaultGatekeeperServiceName  = "gatekeeper-webhook-service"
	DefaultLoggingServiceName     = "elasticsearch-master"
	DefaultLokiServiceName        = "loki"
	DefaultGrafanaServiceName     = "grafana"
	DefaultPrometheusServiceName  = "prometheus-server"
	DefaultKubeappsServiceName    = "kubeapps"

	DefaultRegistryIngressName    = "docker-registry-ingress"
	DefaultChartmuseumIngressName = "chartmuseum-ingress"
	DefaultDashboardIngressName   = "dashboard-ingress"
	DefaultGatekeeperIngressName  = "gatekeeper-ingress"
	DefaultLoggingIngressName     = "logging-ingress"
	DefaultLokiIngressName        = "loki-ingress"
	DefaultGrafanaIngressName     = "grafana-ingress"
	DefaultPrometheusIngressName  = "prometheus-ingress"
	DefaultKubeappsIngressName    = "kubeapps-ingress"

	DefaultRegistryDeploymentName    = "registry-docker-registry"
	DefaultChartmuseumDeploymentName = "chartmuseum-chartmuseum"
	DefaultDashboardDeploymentName   = "dashboard-kubernetes-dashboard"
	DefaultGatekeeperDeploymentName  = "gatekeeper-controller-manager"
	DefaultKubeappsDeploymentName    = "kubeapps"
	DefaultLoggingStateSetsfulName   = "elasticsearch-master"
	DefaultLokiStateSetsfulName      = "loki"
	DefaultGrafanaDeploymentName     = "grafana"
	DefaultPrometheusDeploymentName  = "prometheus-server"

	ClusterHealthLevelError   = "error"
	ClusterHealthLevelWarning = "warning"
	ClusterHealthLevelSuccess = "success"
)
