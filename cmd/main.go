package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/manna-delivery/prometheus-ecs-config-reloader/pkg/aws"
)

const (
	sourceSSM = "ssm"
	sourceS3  = "s3"
)

type ConfigValue interface {
	GetConfigValue() string
}

type S3 struct {
	bucket string
	key    string
}
type SSM struct {
	path string
}

func (s S3) GetConfigValue() string {
	return aws.DownloadObject(s.bucket, s.key)
}

func (s SSM) GetConfigValue() string {
	return *aws.GetParameter(s.path)
}

func loadPrometheusConfig(configValue ConfigValue, configFileDir, configFileName string) {
	prometheusConfig := configValue.GetConfigValue()
	err := ioutil.WriteFile(strings.Join([]string{configFileDir, configFileName}, "/"), []byte(prometheusConfig), 0644)
	if err != nil {
		log.Fatalf("Failed to save config to file: %v", err)
	}
}

func reloadCloudMapScrapeConfig(configValue ConfigValue, configFileDir string) {
	namespaceList := configValue.GetConfigValue()
	namespaces := strings.Split(namespaceList, ",")
	scrapConfig := aws.GetPrometheusScrapeConfig(namespaces)
	tmpfile, err := ioutil.TempFile(configFileDir, "config")
	if err != nil {
		log.Fatalf("Failed creating temp file: %v", err)
	}
	err = ioutil.WriteFile(tmpfile.Name(), []byte(*scrapConfig), 0644)
	if err != nil {
		log.Fatalf("Failed writing to file: %v", err)
	}
	err = os.Rename(tmpfile.Name(), strings.Join([]string{configFileDir, "cloudmap.json"}, "/"))
	if err != nil {
		log.Fatalf("Failed moving to permanent config: %v", err)
	}
}

func main() {
	var prometheusScrapeSourcePath string
	var configValue, scraperValue ConfigValue
	log.Println("Prometheus configuration reloader started")

	configSource, present := os.LookupEnv("CONFIG_SOURCE")
	if !present {
		log.Fatalf("Missing 'CONFIG_SOURCE' environment variable, allowed values: '%s', '%s'\n", sourceSSM, sourceS3)
	}
	prometheusSourcePath, present := os.LookupEnv("SOURCE_PROMETHEUS_CONFIG_FILE_PATH")
	if !present {
		log.Fatalln("Missing 'SOURCE_PROMETHEUS_CONFIG_FILE_PATH' environment variable")
	}
	prometheusScrapeSourcePath, prometheusScrapeSourcePresent := os.LookupEnv("SOURCE_PROMETHEUS_SCRAPE_CONFIGS_PATH")
	if !prometheusScrapeSourcePresent {
		log.Println("Missing 'SOURCE_PROMETHEUS_SCRAPE_CONFIGS_PATH' environment variable")
	}

	configFileDir, present := os.LookupEnv("CONFIG_FILE_DIR")
	if !present {
		configFileDir = "/etc/config/"
		log.Printf("Using default value for 'CONFIG_FILE_DIR': %s", configFileDir)
	}
	configFileName, present := os.LookupEnv("CONFIG_FILE_NAME")
	if !present {
		configFileName = "config.yml"
		log.Printf("Using default value for 'CONFIG_FILE_NAME': %s", configFileName)
	}
	configReloadFrequency, present := os.LookupEnv("CONFIG_RELOAD_FREQUENCY")
	if !present {
		configReloadFrequency = "30"
		log.Printf("Using default value for 'CONFIG_RELOAD_FREQUENCY': %s", configReloadFrequency)
	}

	switch configSource {
	case sourceS3:
		s3ParsedPath, err := url.Parse(prometheusSourcePath)
		if err != nil {
			log.Fatalf("Unable to parse path item %q, %v", prometheusSourcePath, err)
		}
		configValue = S3{
			bucket: s3ParsedPath.Host,
			key:    s3ParsedPath.Path,
		}
		if prometheusScrapeSourcePresent {
			s3ParsedPath, err = url.Parse(prometheusScrapeSourcePath)
			if err != nil {
				log.Fatalf("Unable to parse path item %q, %v", prometheusScrapeSourcePath, err)
			}
			scraperValue = S3{
				bucket: s3ParsedPath.Host,
				key:    s3ParsedPath.Path,
			}
		}
	case sourceSSM:
		configValue = SSM{
			path: path.Clean(prometheusSourcePath),
		}
		if prometheusScrapeSourcePresent {
			scraperValue = SSM{
				path: path.Clean(prometheusScrapeSourcePath),
			}
		}
	default:
		log.Fatalf("Unknown source requested %s\n", configSource)
	}

	aws.InitializeAWSSession()

	loadPrometheusConfig(configValue, configFileDir, configFileName)
	log.Println("Loaded initial configuration file")
	if prometheusScrapeSourcePresent {
		log.Println("Found scrapers configuration")
		reloadCloudMapScrapeConfig(scraperValue, configFileDir)

		if configReloadFrequency != "0" {
			log.Println("Starting daemon reloader mode")
			go func() {
				reloadFrequency, _ := strconv.Atoi(configReloadFrequency)
				ticker := time.NewTicker(time.Duration(reloadFrequency) * time.Second)
				for {
					<-ticker.C
					reloadCloudMapScrapeConfig(scraperValue, configFileDir)
					log.Println("Reloaded CloudMap config")
				}
			}()
			log.Println("Periodic reloads under progress...")

			stopChannel := make(chan string)
			for {
				status := <-stopChannel
				fmt.Println(status)
				break
			}
		}
	}
}
