package config

import (
	"bytes"
	"fmt"
	"geeksaga.com/os/straw/internal"
	"geeksaga.com/os/straw/internal/models"
	"geeksaga.com/os/straw/plugins/inputs"
	"geeksaga.com/os/straw/plugins/outputs"
	serializers "geeksaga.com/os/straw/plugins/serializers"
	"github.com/influxdata/toml"
	"github.com/influxdata/toml/ast"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	// Default sections
	sectionDefaults = []string{"global_tags", "agent", "outputs", "processors", "inputs"}

	// Default input plugins
	inputDefaults = []string{"cpu", "mem", "swap", "system", "kernel", "processes", "disk", "diskio"}

	// Default output plugins
	outputDefaults = []string{"json"}

	// envVarRe is a regex to find environment variables in the config file
	envVarRe = regexp.MustCompile(`\$\{(\w+)\}|\$(\w+)`)

	envVarEscaper = strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
	)
)

type Config struct {
	Tags map[string]string

	Agent   *AgentConfig
	Inputs  []*models.RunningInput
	Outputs []*models.RunningOutput
}

type AgentConfig struct {
	// Interval at which to gather information
	Interval internal.Duration
	// RoundInterval rounds collection interval to 'interval'.
	//     ie, if Interval=10s then always collect on :00, :10, :20, etc.
	RoundInterval    bool
	Precision        internal.Duration
	FlushInterval    internal.Duration
	CollectionJitter internal.Duration
	FlushJitter      internal.Duration

	MetricBatchSize   int
	MetricBufferLimit int

	// Quiet is the option for running in quiet mode
	Quiet bool `toml:"quiet"`

	LogTarget string `toml:"logtarget"`
	Logfile   string `toml:"logfile"`

	LogfileRotationInterval    internal.Duration `toml:"logfile_rotation_interval"`
	LogfileRotationMaxSize     internal.Size     `toml:"logfile_rotation_max_size"`
	LogfileRotationMaxArchives int               `toml:"logfile_rotation_max_archives"`

	Hostname     string
	OmitHostname bool
}

func (c *Config) InputNames() []string {
	var name []string
	for _, input := range c.Inputs {
		name = append(name, input.Config.Name)
	}
	return name
}

func (c *Config) OutputNames() []string {
	var name []string
	for _, output := range c.Outputs {
		name = append(name, output.Config.Name)
	}
	return name
}

func (c *Config) ListTags() string {
	var tags []string

	for k, v := range c.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(tags)

	return strings.Join(tags, " ")
}

func NewConfig() *Config {
	c := &Config{
		// Agent defaults:
		Agent: &AgentConfig{
			Interval:      internal.Duration{Duration: 10 * time.Second},
			RoundInterval: true,
			FlushInterval: internal.Duration{Duration: 10 * time.Second},
			LogTarget:     "file",
		},

		Tags:    make(map[string]string),
		Inputs:  make([]*models.RunningInput, 0),
		Outputs: make([]*models.RunningOutput, 0),
	}
	return c
}

func (c *Config) LoadDirectory(path string) error {
	walkfn := func(thispath string, info os.FileInfo, _ error) error {
		if info == nil {
			log.Printf("W! %s is not permitted to read %s", "Straw", thispath)
			return nil
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), "..") {
				// skip Kubernetes mounts, prevening loading the same config twice
				return filepath.SkipDir
			}

			return nil
		}
		name := info.Name()
		if len(name) < 6 || name[len(name)-5:] != ".conf" {
			return nil
		}
		err := c.LoadConfig(thispath)
		if err != nil {
			return err
		}
		return nil
	}
	return filepath.Walk(path, walkfn)
}

// Try to find a default config file at these locations (in order):
//   1. $STRAW_CONFIG_PATH
//   2. $HOME/.straw/straw.conf
//   3. /etc/straw/straw.conf
func getDefaultConfigPath() (string, error) {
	envfile := os.Getenv("STRAW_CONFIG_PATH")
	homefile := os.ExpandEnv("${HOME}/.straw/straw.conf")
	etcfile := "/etc/straw/straw.conf"

	for _, path := range []string{envfile, homefile, etcfile} {
		if _, err := os.Stat(path); err == nil {
			log.Printf("I! Using config file: %s", path)
			return path, nil
		}
	}

	// if we got here, we didn't find a file in a default location
	return "", fmt.Errorf("No config file specified, and could not find one"+
		" in $STRAW_CONFIG_PATH, %s, or %s", homefile, etcfile)
}

func (c *Config) LoadConfig(path string) error {
	var err error
	if path == "" {
		if path, err = getDefaultConfigPath(); err != nil {
			return err
		}
	}
	data, err := loadConfig(path)
	if err != nil {
		return fmt.Errorf("Error loading %s, %s", path, err)
	}

	tbl, err := parseConfig(data)
	if err != nil {
		return fmt.Errorf("Error parsing %s, %s", path, err)
	}

	// Parse tags tables first:
	for _, tableName := range []string{"tags", "global_tags"} {
		if val, ok := tbl.Fields[tableName]; ok {
			subTable, ok := val.(*ast.Table)
			if !ok {
				return fmt.Errorf("%s: invalid configuration", path)
			}
			if err = toml.UnmarshalTable(subTable, c.Tags); err != nil {
				log.Printf("E! Could not parse [global_tags] config\n")
				return fmt.Errorf("Error parsing %s, %s", path, err)
			}
		}
	}

	// Parse agent table:
	if val, ok := tbl.Fields["agent"]; ok {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}
		if err = toml.UnmarshalTable(subTable, c.Agent); err != nil {
			log.Printf("E! Could not parse [agent] config\n")
			return fmt.Errorf("Error parsing %s, %s", path, err)
		}
	}

	if !c.Agent.OmitHostname {
		if c.Agent.Hostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}

			c.Agent.Hostname = hostname
		}

		c.Tags["host"] = c.Agent.Hostname
	}

	// Parse all the rest of the plugins:
	for name, val := range tbl.Fields {
		subTable, ok := val.(*ast.Table)
		if !ok {
			return fmt.Errorf("%s: invalid configuration", path)
		}

		switch name {
		case "agent", "global_tags", "tags":
		case "outputs":
			for pluginName, pluginVal := range subTable.Fields {
				switch pluginSubTable := pluginVal.(type) {
				case []*ast.Table:
					for _, t := range pluginSubTable {
						if err = c.addOutput(pluginName, t); err != nil {
							return fmt.Errorf("Error parsing %s, %s", path, err)
						}
					}
				default:
					return fmt.Errorf("Unsupported config format: %s, file %s",
						pluginName, path)
				}
			}
		case "inputs", "plugins":
			for pluginName, pluginVal := range subTable.Fields {
				switch pluginSubTable := pluginVal.(type) {
				case []*ast.Table:
					for _, t := range pluginSubTable {
						if err = c.addInput(pluginName, t); err != nil {
							return fmt.Errorf("Error parsing %s, %s", path, err)
						}
					}
				default:
					return fmt.Errorf("Unsupported config format: %s, file %s", pluginName, path)
				}
			}
		// Assume it's an input input for legacy config file support if no other identifiers are present
		default:
			if err = c.addInput(name, subTable); err != nil {
				return fmt.Errorf("Error parsing %s, %s", path, err)
			}
		}
	}

	return nil
}

// trimBOM trims the Byte-Order-Marks from the beginning of the file.
// this is for Windows compatibility only.
// see https://github.com/influxdata/telegraf/issues/1378
func trimBOM(f []byte) []byte {
	return bytes.TrimPrefix(f, []byte("\xef\xbb\xbf"))
}

// escapeEnv escapes a value for inserting into a TOML string.
func escapeEnv(value string) string {
	return envVarEscaper.Replace(value)
}

func loadConfig(config string) ([]byte, error) {
	u, err := url.Parse(config)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https", "http":
		return fetchConfig(u)
	default:
		// If it isn't a https scheme, try it as a file.
	}
	return ioutil.ReadFile(config)
}

func fetchConfig(u *url.URL) ([]byte, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	if v, exists := os.LookupEnv("STRAW_TOKEN"); exists {
		req.Header.Add("Authorization", "Token "+v)
	}
	req.Header.Add("Accept", "application/toml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve remote config: %s", resp.Status)
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// parseConfig loads a TOML configuration from a provided path and
// returns the AST produced from the TOML parser. When loading the file, it
// will find environment variables and replace them.
func parseConfig(contents []byte) (*ast.Table, error) {
	contents = trimBOM(contents)

	parameters := envVarRe.FindAllSubmatch(contents, -1)
	for _, parameter := range parameters {
		if len(parameter) != 3 {
			continue
		}

		var env_var []byte
		if parameter[1] != nil {
			env_var = parameter[1]
		} else if parameter[2] != nil {
			env_var = parameter[2]
		} else {
			continue
		}

		env_val, ok := os.LookupEnv(strings.TrimPrefix(string(env_var), "$"))
		if ok {
			env_val = escapeEnv(env_val)
			contents = bytes.Replace(contents, parameter[0], []byte(env_val), 1)
		}
	}

	return toml.Parse(contents)
}

func (c *Config) addInput(name string, table *ast.Table) error {
	creator, ok := inputs.Inputs[name]
	if !ok {
		return fmt.Errorf("undefined but requested input: %s", name)
	}
	input := creator()

	pluginConfig, err := buildInput(name, table)
	if err != nil {
		return err
	}

	if err := toml.UnmarshalTable(table, input); err != nil {
		return err
	}

	rp := models.NewRunningInput(input, pluginConfig)
	rp.SetDefaultTags(c.Tags)
	c.Inputs = append(c.Inputs, rp)

	return nil
}

func (c *Config) addOutput(name string, table *ast.Table) error {
	creator, ok := outputs.Outputs[name]
	if !ok {
		return fmt.Errorf("undefined but requested output: %s", name)
	}
	output := creator()

	// If the output has a SetSerializer function, then this means it can write
	// arbitrary types of output, so build the serializer and set it.
	switch t := output.(type) {
	case serializers.SerializerOutput:
		serializer, err := buildSerializer(name, table)

		if err != nil {
			return err
		}
		t.SetSerializer(serializer)
	}

	outputConfig, err := buildOutput(name, table)
	if err != nil {
		return err
	}

	if err := toml.UnmarshalTable(table, output); err != nil {
		return err
	}

	ro := models.NewRunningOutput(name, output, outputConfig,
		c.Agent.MetricBatchSize, c.Agent.MetricBufferLimit)
	c.Outputs = append(c.Outputs, ro)
	return nil
}

func buildInput(name string, tbl *ast.Table) (*models.InputConfig, error) {
	cp := &models.InputConfig{Name: name}
	if node, ok := tbl.Fields["interval"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				dur, err := time.ParseDuration(str.Value)
				if err != nil {
					return nil, err
				}

				cp.Interval = dur
			}
		}
	}

	if node, ok := tbl.Fields["name_prefix"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				cp.MeasurementPrefix = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["name_suffix"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				cp.MeasurementSuffix = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["name_override"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				cp.NameOverride = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["alias"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				cp.Alias = str.Value
			}
		}
	}

	cp.Tags = make(map[string]string)
	if node, ok := tbl.Fields["tags"]; ok {
		if subtbl, ok := node.(*ast.Table); ok {
			if err := toml.UnmarshalTable(subtbl, cp.Tags); err != nil {
				log.Printf("E! Could not parse tags for input %s\n", name)
			}
		}
	}

	delete(tbl.Fields, "name_prefix")
	delete(tbl.Fields, "name_suffix")
	delete(tbl.Fields, "name_override")
	delete(tbl.Fields, "alias")
	delete(tbl.Fields, "interval")
	delete(tbl.Fields, "tags")

	return cp, nil
}

// buildOutput parses output specific items from the ast.Table,
// builds the filter and returns an
// models.OutputConfig to be inserted into models.RunningInput
// Note: error exists in the return for future calls that might require error
func buildOutput(name string, tbl *ast.Table) (*models.OutputConfig, error) {
	oc := &models.OutputConfig{
		Name: name,
	}

	if node, ok := tbl.Fields["flush_interval"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				dur, err := time.ParseDuration(str.Value)
				if err != nil {
					return nil, err
				}

				oc.FlushInterval = dur
			}
		}
	}

	if node, ok := tbl.Fields["flush_jitter"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				dur, err := time.ParseDuration(str.Value)
				if err != nil {
					return nil, err
				}
				oc.FlushJitter = new(time.Duration)
				*oc.FlushJitter = dur
			}
		}
	}

	if node, ok := tbl.Fields["metric_buffer_limit"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if integer, ok := kv.Value.(*ast.Integer); ok {
				v, err := integer.Int()
				if err != nil {
					return nil, err
				}
				oc.MetricBufferLimit = int(v)
			}
		}
	}

	if node, ok := tbl.Fields["metric_batch_size"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if integer, ok := kv.Value.(*ast.Integer); ok {
				v, err := integer.Int()
				if err != nil {
					return nil, err
				}
				oc.MetricBatchSize = int(v)
			}
		}
	}

	if node, ok := tbl.Fields["alias"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				oc.Alias = str.Value
			}
		}
	}

	delete(tbl.Fields, "flush_interval")
	delete(tbl.Fields, "flush_jitter")
	delete(tbl.Fields, "metric_buffer_limit")
	delete(tbl.Fields, "metric_batch_size")
	delete(tbl.Fields, "alias")

	return oc, nil
}

// buildSerializer grabs the necessary entries from the ast.Table for creating
// a serializers.Serializer object, and creates it, which can then be added onto
// an Output object.
func buildSerializer(name string, tbl *ast.Table) (serializers.Serializer, error) {
	c := &serializers.Config{TimestampUnits: time.Duration(1 * time.Second)}

	if node, ok := tbl.Fields["data_format"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				c.DataFormat = str.Value
			}
		}
	}

	if node, ok := tbl.Fields["json_timestamp_units"]; ok {
		if kv, ok := node.(*ast.KeyValue); ok {
			if str, ok := kv.Value.(*ast.String); ok {
				timestampVal, err := time.ParseDuration(str.Value)
				if err != nil {
					return nil, fmt.Errorf("Unable to parse json_timestamp_units as a duration, %s", err)
				}
				// now that we have a duration, truncate it to the nearest
				// power of ten (just in case)
				nearest_exponent := int64(math.Log10(float64(timestampVal.Nanoseconds())))
				new_nanoseconds := int64(math.Pow(10.0, float64(nearest_exponent)))
				c.TimestampUnits = time.Duration(new_nanoseconds)
			}
		}
	}

	delete(tbl.Fields, "data_format")
	delete(tbl.Fields, "json_timestamp_units")

	return serializers.NewSerializer(c)
}
