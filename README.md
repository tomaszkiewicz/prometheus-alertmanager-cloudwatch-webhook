# Prometheus Alertmanager Cloudwatch Webhook

The project provides application that exposes webhook to be used by Prometheus Alertmanager.
The webhook can be invoked by DeadMansSwitch alert from Prometheus and on every invocation it updates a metric in CloudWatch.
You can use that metric to setup Cloudwatch Alert and get notified when Prometheus alerting pipeline is not healthy.

This project can also create multiple Cloudwatch Alerts. It uses the alert name from the Alertmanager webhook as the metric name.

You can find more information [in this blog post](https://luktom.net/en/e1629-monitoring-prometheus-alerting-pipeline-health-using-cloudwatch).

## Requirements

1. IAM Role (to allow the app to put CloudWatch metrics)
2. kube2iam (as default deployment uses kube2iam annotation)
3. kustomize tool (or you can apply manifests manually)

## Installation

### Create required IAM role

Create new IAM role named _k8s-alertmanager-cloudwatch-webhook_ with the following trust relationship:

```json
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "AWS": [
          "arn:aws:iam::123456789012:role/nodes.cluster.name"
        ]
      },
      "Effect": "Allow"
    }
  ]
}
```

Replace principal ARN with the role your Kubernetes nodes use.
Then set the policy to:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "cloudwatch:PutMetricData"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

### Deploy application to kubernetes

```sh
git clone git@github.com:tomaszkiewicz/prometheus-alertmanager-cloudwatch-webhook.git
cd prometheus-alertmanager-cloudwatch-webhook/build/k8s
kustomize build | kubectl apply -f -
```

If you do not use kustomize you can apply deployment.yaml and service.yaml files manually.
Please note that by default configuration is to deploy to _monitoring_ namespace as it's also the case for Prometheus Operator.

### Configure Alertmanager

Modify your configuration to include new receiver definition and new route. Here's sample configuration:

```yaml
route:
  receiver: "slack"
  group_by:
  - severity
  - alertname
  routes:
  - receiver: "cloudwatch"
    match:
      alertname: DeadMansSwitch
    group_wait: 30s
    group_interval: 1m
    repeat_interval: 1m
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 48h

receivers:
- name: "cloudwatch"
  webhook_configs:
  - url: "http://alertmanager-cloudwatch-webhook/webhook"
- name: "slack"
  ...
```

### Configure CloudWatch Alert

Example alert definition in Terraform:

```terraform
resource "aws_cloudwatch_metric_alarm" "p8s_dead_mans_switch" {
  alarm_name = "prometheus-alertmanager-pipeline-health"
  alarm_description = "This metric shows health of alerting pipeline"
  comparison_operator = "LessThanThreshold"
  evaluation_periods = "5"
  metric_name = "DeadMansSwitch"
  namespace = "Prometheus"
  period = "60"
  statistic = "Minimum"
  threshold = "1"
  treat_missing_data = "breaching"
  alarm_actions = ["${module.slack_alarm_notification.sns_topic_arn}"]
  ok_actions = ["${module.slack_alarm_notification.sns_topic_arn}"]
}
```

You should modify _alarm_actions_ and _ok_actions_ and set it to your CloudWatch alerting system.

## Configuration

### Environment variables

You can configure some aspect of the webhook via environment variables.

| Variable    | Default Value | Description                                          |
| ----------- | ------------- | ---------------------------------------------------- |
| `HTTP_PORT` | 8077          | the port on which the webhook listen                 |
| `REGION`    | eu-west-1     | the AWS region where the metric will be created      |
| `NAMESPACE` | Prometheus    | the namespace under which the metric will be created |

### Config file

You can optionally mount a `config.yml` file into the config folder `/config`.
This file follow the following pattern

```yml
http-port: 8077
region: eu-west-1
namespace: Prometheus
dimensions:
  alerts:
    myAlertName:            # you can explicitly include labels for a particular alert.
      - aLabel              # labels can be explicitly listed to be converted to dimensions
      - anotherLabel
```

### Metric dimensions

You can map alerts' labels into metrics' dimensions from the mapping declaration in the `config.yml` file.
