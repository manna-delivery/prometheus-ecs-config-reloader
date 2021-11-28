package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/manna-delivery/prometheus-ecs-config-reloader/pkg/aws"
)

func loadPrometheusConfig(prometheusSSMPath, configFileDir string) {
	prometheusConfig := aws.GetParametersByPath(path.Clean(prometheusSSMPath))
	prometheusConfigExists := false
	for _, config := range prometheusConfig {
		if (filepath.Base(*config.Name) == "prometheus.yml") || (filepath.Base(*config.Name) == "prometheus.yaml") {
			prometheusConfigExists = true
		}
		err := ioutil.WriteFile(strings.Join([]string{configFileDir, filepath.Base(*config.Name)}, "/"), []byte(*config.Value), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	if !prometheusConfigExists {
		log.Fatalf("Prometheus main configuration file('prometheus.yml' OR 'prometheus.yaml') value was not found in SSM path '%s'\n", prometheusSSMPath)
	}
}

func reloadCloudMapScrapeConfig(prometheusScrapeSSMPath, configFileDir string) {
	namespaceList := aws.GetParameter(strings.Join([]string{path.Clean(prometheusScrapeSSMPath), "cloudmap"}, "/"))
	namespaces := strings.Split(*namespaceList, ",")
	scrapConfig := aws.GetPrometheusScrapeConfig(namespaces)
	err := ioutil.WriteFile(strings.Join([]string{configFileDir, "cloudmap.json"}, "/"), []byte(*scrapConfig), 0644)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	log.Println("Prometheus configuration reloader started")

	prometheusSSMPath, present := os.LookupEnv("SSM_PROMETHEUS_CONFIG_FILES_PATH")
	if !present {
		log.Fatalln("Missing 'SSM_PROMETHEUS_CONFIG_FILES_PATH' environment variable")
	}
	prometheusScrapeSSMPath, present := os.LookupEnv("SSM_PROMETHEUS_SCRAPE_CONFIGS_PATH")
	if !present {
		log.Fatalln("Missing 'SSM_PROMETHEUS_SCRAPE_CONFIGS_PATH' environment variable")
	}

	aws.InitializeAWSSession()

	configFileDir, present := os.LookupEnv("CONFIG_FILE_DIR")
	if !present {
		configFileDir = "/etc/config/"
		log.Printf("Using default value for 'CONFIG_FILE_DIR': %s", configFileDir)
	}
	configReloadFrequency, present := os.LookupEnv("CONFIG_RELOAD_FREQUENCY")
	if !present {
		configReloadFrequency = "30"
		log.Printf("Using default value for 'CONFIG_RELOAD_FREQUENCY': %s", configReloadFrequency)
	}

	loadPrometheusConfig(prometheusSSMPath, configFileDir)
	reloadCloudMapScrapeConfig(prometheusScrapeSSMPath, configFileDir)
	log.Println("Loaded initial configuration file")

	go func() {
		reloadFrequency, _ := strconv.Atoi(configReloadFrequency)
		ticker := time.NewTicker(time.Duration(reloadFrequency) * time.Second)
		for {
			<-ticker.C
			reloadCloudMapScrapeConfig(prometheusScrapeSSMPath, configFileDir)
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
