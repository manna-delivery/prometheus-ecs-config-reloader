# Prometheus ECS Config Reloader

Side car service for Prometheus Server running in ECS that pull configurations from AWS SSM Parameter Store and reload the configuration files

## Motivation

The code is based on original AWS example how to run Prometheus Server in ECS, with few modification to support internal requirements

## Configurations

| Name | Description | Default Value |
|---|---|---|
| `SSM_PROMETHEUS_CONFIG_FILES_PATH` | SSM Parameters path that include all Prometheus configuration files | None |
| `SSM_PROMETHEUS_SCRAPE_CONFIGS_PATH` | SSM Parameters path that include all Prometheus scrapers configurations | None |
| `CONFIG_FILE_DIR` | Local directory to store configuration from SSM | `/etc/config/` |
| `CONFIG_RELOAD_FREQUENCY` | Frequency to reload scrapers configurations from SSM | `30` |

### Prometheus Configuration files

The service support multiple configuration files to be loaded from SSM for better management.  
Each paramater under `SSM_PROMETHEUS_CONFIG_FILES_PATH` path is a file, where the SSM Paramater name is the file name.  
At least one configuration file must be created in SSM path, the main file - `prometheus.yml`. All other files if used must be included inside of it (Certificates, password files and etc).  

### Prometheus Scrape Configurations

#### AWS CloudMap

Scrape services registred in AWS CloudMap namespaces. Create under `SSM_PROMETHEUS_SCRAPE_CONFIGS_PATH` SSM paramater named `cloudmap` and add comma separated list of CloudMap namespaces to scrape services from.

Add the following configuration to Prometheus configuration if using CloudMap scraping:
```yaml
scrape_configs:
  - job_name: cloudmap
    file_sd_configs:
      - files:
          - /etc/config/cloudmap.json
        refresh_interval: 30s
```

### Example configuration

- SSM Parameter: `/dev/prometheus/files/prometheus.yml`  
    Value:
    ```yaml
    global:
    evaluation_interval: 1m
    scrape_interval: 30s
    scrape_timeout: 10s
    scrape_configs:
    - job_name: cloudmap
        file_sd_configs:
        - files:
            - /etc/config/cloudmap.json
            refresh_interval: 30s
    ```
- SSM Parameter: `/dev/prometheus/scrapers/cloudmap`  
    Value: `ecs-services`

## Disclaimer

The project is not affiliated with AWS or Prometheus.
