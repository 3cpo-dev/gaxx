package providers

type Config struct {
	Providers struct {
		Default string `yaml:"default"`
		Linode  struct {
			Token  string   `yaml:"token"`
			Region string   `yaml:"region"`
			Type   string   `yaml:"type"`
			Image  string   `yaml:"image"`
			Tags   []string `yaml:"tags"`
		} `yaml:"linode"`
		Vultr struct {
			Token  string   `yaml:"token"`
			Region string   `yaml:"region"`
			Plan   string   `yaml:"plan"`
			OSID   string   `yaml:"os_id"`
			Tags   []string `yaml:"tags"`
		} `yaml:"vultr"`
		LocalSSH struct {
			Hosts []struct {
				Name    string `yaml:"name"`
				IP      string `yaml:"ip"`
				User    string `yaml:"user"`
				KeyPath string `yaml:"key_path"`
				Port    int    `yaml:"port"`
			} `yaml:"hosts"`
		} `yaml:"localssh"`
	} `yaml:"providers"`
	SSH struct {
		KeyDir     string `yaml:"key_dir"`
		KnownHosts string `yaml:"known_hosts"`
	} `yaml:"ssh"`
	Defaults struct {
		User           string `yaml:"user"`
		SSHPort        int    `yaml:"ssh_port"`
		Retries        int    `yaml:"retries"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
	} `yaml:"defaults"`
	Telemetry struct {
		Enabled         bool   `yaml:"enabled"`
		OTLPEndpoint    string `yaml:"otlp_endpoint"`
		MonitoringPort  int    `yaml:"monitoring_port"`
		MetricsInterval int    `yaml:"metrics_interval"`
	} `yaml:"telemetry"`
}
