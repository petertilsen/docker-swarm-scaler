# Docker Swarm Scaler as golang package

[![](https://images.microbadger.com/badges/image/petertilsen1/docker-swarm-scaler.svg)](https://microbadger.com/images/petertilsen1/docker-swarm-scaler "Get your own image badge on microbadger.com")
[![](https://images.microbadger.com/badges/version/petertilsen1/docker-swarm-scaler.svg)](https://microbadger.com/images/petertilsen1/docker-swarm-scaler "Get your own version badge on microbadger.com")
[![](https://images.microbadger.com/badges/commit/petertilsen1/docker-swarm-scaler.svg)](https://microbadger.com/images/petertilsen1/docker-swarm-scaler "Get your own commit badge on microbadger.com")
[![Build Status](https://travis-ci.org/petertilsen/docker-swarm-scaler.svg?branch=master)](https://travis-ci.org/petertilsen/docker-swarm-scaler)
[![Coverage Status](https://coveralls.io/repos/github/petertilsen/docker-swarm-scaler/badge.svg?branch=master)](https://coveralls.io/github/petertilsen/docker-swarm-scaler?branch=master)

## Features ##

* Up and Down scaling of docker swarm services
* AWS ECR access to update distributed swarm images

## Prerequisites

### Prometheus ### 

When setting your alerts with Prometheus introduce an additional annotation for **service** into your rules file

```yaml
service: "{{ $labels.container_label_com_docker_swarm_service_name }}"
```

Full example
``` yaml
groups:
- name: containers
  rules:
  - alert: container_eating_memory
    expr: sum(container_memory_rss{container_label_com_docker_swarm_task_name=~".+"}) by (container_label_com_docker_swarm_service_name,instance,image) / 1000000 > 3000
    for: 1m
    labels:
      severity: critical
    annotations:
      service: "{{ $labels.container_label_com_docker_swarm_service_name }}"
      summary: "Container {{ $labels.container_label_com_docker_swarm_service_name }} has high memory consumption"
      description: "{{ $labels.name }} with image {{ $labels.image }} of service {{ $labels.container_label_com_docker_swarm_service_name }} on {{ $labels.instance }} is eating up memory. Usage is {{ humanize $value}}"

```

### Alertmanager ###

In order to feed the scaler with alerts you need to introduce a webhook receiver

```yaml
route:
  receiver: 'receiver'

receivers:
  - name: 'receiver'
    webhook_configs:
        - send_resolved: true
          url: http://[SCALER URL]:8083
```


Alert json payload from Alertmanger will contain the name of your service

```json
{
"annotations": {"service": "[YOUR DOCKER SWARM SERVICE]", "summary": "test", "description": "test"}
}
```

Full example of valid Alertmanger Webhook json payload

```json
{
  "version": "4",
  "groupKey": "1",
  "status": "firing",
  "alerts": [
    {
      "labels": {"summary": "test", "description": "test"},
      "annotations": {"service": "YOUR DOCKER SWARM SERVICE", "summary": "test", "description": "test"},
      "startsAt": "2012-11-01T22:08:41+00:00",
      "endsAt": "2012-11-01T22:08:41+00:00"
    }
  ]
}

```

## Run in docker compose 

```docker-compose
version: '3.4'

services:
  scaler_dev:
    image: petertilsen1/docker-tools_scaler:latest
    envorinment:
        - AWS_ACCESS_KEY_ID=[YOUR ID]
        - AWS_SECRET_ACCESS_KEY=[YOUR ACCESS KEY]
    ports:
      - "8083:8083"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: replicated
      replicas: 1
      restart_policy:
        condition: any
    logging:
      driver: json-file

```

## Authors ##
Scaler was created by [Peter Tilsen](https://github.com/petertilsen)