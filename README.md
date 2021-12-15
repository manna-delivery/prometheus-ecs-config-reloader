# Prometheus ECS Config Reloader

Side car service for Prometheus Server / Exporters running in ECS that pull configurations from AWS SSM Parameter Store / S3 and reload the configuration files

## Motivation

The code is based on [original AWS example](https://github.com/aws-samples/prometheus-for-ecs/) how to run Prometheus Server in ECS, with few modification to support internal requirements

## Configurations

| Name | Description | Default Value |
|---|---|---|
| `CONFIG_SOURCE` | Config source to pull configurations from | None [`s3`, `ssm`] |
| `SOURCE_PROMETHEUS_CONFIG_FILE_PATH` | Path for SSM Parameter or S3 key path that include all Prometheus configuration files. S3 Path should start with `s3://` protocl and include bucket and full path | None |
| `SOURCE_PROMETHEUS_SCRAPE_CONFIGS_PATH` | Path for SSM Parameter path or S3 key that include all Prometheus scrapers configurations. This config can be reloaded automatically | None (optional) |
| `CONFIG_FILE_DIR` | Local directory to store configuration from SSM / S3 | `/etc/config/` |
| `CONFIG_FILE_NAME` | Local config file name | `config.yml` |
| `CONFIG_RELOAD_FREQUENCY` | Frequency to reload scrapers configurations from SSM / S3. `0` disables this functionality | `30` |

### Configuration Files in ECS

Prometheus ECS Config Reloader is built to run as side car for Prometheus Server or Exproter container in ECS with shared volume between both containers.

#### Prometheus Server Example Task Definition:

```json
"containerDefinitions": [
  {
    "cpu": 128,
    "environment": [
      {
        "name": "AWS_REGION",
        "value": "eu-west-1"
      },
      {
        "name": "CONFIG_FILE_DIR",
        "value": "/etc/config"
      },
      {
        "name": "CONFIG_FILE_NAME",
        "value": "prometheus.yaml"
      },
      {
        "name": "CONFIG_RELOAD_FREQUENCY",
        "value": "60"
      },
      {
        "name": "CONFIG_SOURCE",
        "value": "ssm"
      },
      {
        "name": "SOURCE_PROMETHEUS_CONFIG_FILE_PATH",
        "value": "/prometheus/files/prometheus.yaml"
      },
      {
        "name": "SOURCE_PROMETHEUS_SCRAPE_CONFIGS_PATH",
        "value": "/prometheus/scrapers/cloudmap"
      }
    ],
    "mountPoints": [
      {
        "readOnly": false,
        "containerPath": "/etc/config",
        "sourceVolume": "configVolume"
      }
    ],
    "memory": 128,
    "image": "public.ecr.aws/manna/prometheus-ecs-config-reloader:1.1.0",
    "essential": false,
    "user": "root",
    "name": "config-reloader"
  },
  {
    "portMappings": [
      {
        "hostPort": 9090,
        "protocol": "tcp",
        "containerPort": 9090
      }
    ],
    "command": [
      "--storage.tsdb.retention.time=15d",
      "--config.file=/etc/config/prometheus.yaml",
      "--storage.tsdb.path=/data",
      "--web.console.libraries=/etc/prometheus/console_libraries",
      "--web.console.templates=/etc/prometheus/consoles",
      "--web.enable-lifecycle"
    ],
    "cpu": 512,
    "mountPoints": [
      {
        "readOnly": null,
        "containerPath": "/etc/config",
        "sourceVolume": "configVolume"
      },
      {
        "readOnly": null,
        "containerPath": "/data",
        "sourceVolume": "logsVolume"
      }
    ],
    "memory": 512,
    "image": "quay.io/prometheus/prometheus:v2.31.2",
    "dependsOn": [
      {
        "containerName": "config-reloader",
        "condition": "START"
      }
    ],
    "healthCheck": {
      "retries": 2,
      "command": [
        "CMD-SHELL",
        "wget http://localhost:9090/-/healthy -O /dev/null|| exit 1"
      ],
      "timeout": 2,
      "interval": 10,
      "startPeriod": 10
    },
    "essential": true,
    "user": "root",
    "name": "prometheus-server"
  }
]
```

#### Prometheus Cloudwatch Exporter Example Task Definition:

```json
"containerDefinitions": [
  {
    "cpu": 128,
    "environment": [
      {
        "name": "AWS_REGION",
        "value": "eu-west-1"
      },
      {
        "name": "CONFIG_FILE_DIR",
        "value": "/config"
      },
      {
        "name": "CONFIG_FILE_NAME",
        "value": "config.yml"
      },
      {
        "name": "CONFIG_RELOAD_FREQUENCY",
        "value": "0"
      },
      {
        "name": "CONFIG_SOURCE",
        "value": "s3"
      },
      {
        "name": "SOURCE_PROMETHEUS_CONFIG_FILE_PATH",
        "value": "s3://artifacts.manna/dev/prometheus/cloudwatch_exporter/config.yaml"
      }
    ],
    "mountPoints": [
      {
        "readOnly": false,
        "containerPath": "/config",
        "sourceVolume": "configVolume"
      }
    ],
    "memory": 128,
    "image": "public.ecr.aws/manna/prometheus-ecs-config-reloader:1.1.0",
    "essential": false,
    "user": "root",
    "name": "config-reloader"
  },
  {
    "portMappings": [
      {
        "hostPort": 9106,
        "protocol": "tcp",
        "containerPort": 9106
      }
    ],
    "cpu": 512,
    "mountPoints": [
      {
        "readOnly": null,
        "containerPath": "/config",
        "sourceVolume": "configVolume"
      }
    ],
    "memory": 512,
    "image": "prom/cloudwatch-exporter:v0.12.2",
    "dependsOn": [
      {
        "containerName": "config-reloader",
        "condition": "SUCCESS"
      }
    ],
    "essential": true,
    "hostname": "prometheus-cloudwatch-exporter",
    "name": "prometheus-cloudwatch_exporter"
  }
]
```

### Configuration Sources

The config fles source can be SSM Paramter Store or S3 file. In case when there is main configuration file and scraper configuration, only one source can be used
### Prometheus Configuration file

Configuration file can be collected from SSM Parameter, this is simplier method, but limited to 4KB (or 8KB in advanced mode). When the size is bigger S3 file can be used to load the configurations, by specifing full S3 URL to the config file: `s3://BUCKET_NAME/KEY/PATH.yml`.

Config file name will be taken from `CONFIG_FILE_NAME` variable and not the actual file name / SSM parameter name.

### Prometheus Scrape Configurations

#### AWS CloudMap

Scrape services registred in AWS CloudMap namespaces. Create under `SOURCE_PROMETHEUS_SCRAPE_CONFIGS_PATH` SSM paramater named `cloudmap` and add comma separated list of CloudMap namespaces to scrape services from.

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
