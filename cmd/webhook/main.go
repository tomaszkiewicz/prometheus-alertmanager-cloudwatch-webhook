package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	HttpPort	int 			`koanf:"http-port"`
	Region		string
	Namespace	string
	Dimension   ConfigDimension
}

type ConfigDimension struct {
	Alerts map[string][]string
}

// Webhook - The main structure for prometheus
// alerts using webhooks
type Webhook struct {
	Receiver          string
	Status            string
	Alerts            []Alerts
	Grouplabels       Grouplabels
	CommonLabels      Labels
	CommonAnnotations Annotations
	ExternalURL       string
	Version           string
	groupKey          string
}

// Alerts - A list of alerts from prometheus
type Alerts struct {
	Status       string
	Labels       map[string]string
	Annotations  Annotations
	StartsAt     string
	EndsAt       string
	GeneratorURL string
}

// Labels - Contains the labels for an alert
type Labels struct {
	Alertname string
}

// Annotations - Contains the annotations for an alert
type Annotations struct {
	Description string
	Summary     string
}

// Grouplabels - The groupLabels field from the webhook
type Grouplabels struct {
	AlertName string
}

var sess *session.Session
var svc *cloudwatch.CloudWatch
var config Config

func webhook(c *gin.Context) {
	if err := putMetric(c); err != nil {
		log.Println("unable to put metric: ", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, struct{}{})
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, struct{}{})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.POST("/webhook", webhook)

	return r
}

func getDimensions(labels *map[string]string) []*cloudwatch.Dimension {
	if alertname, ok := (*labels)["alertname"]; ok {
		dimensionNames := config.Dimension.Alerts[alertname]
		dimensions := make([]*cloudwatch.Dimension, 0, len(dimensionNames))
		for _, name := range dimensionNames {
			if v, ok := (*labels)[name]; ok {
				dimensions = append(dimensions, &cloudwatch.Dimension{Name: aws.String(name), Value: aws.String(v)})
			}
		}

		return dimensions
	}

	return []*cloudwatch.Dimension{}
}

func putMetric(c *gin.Context) error {
	var webhookData Webhook
	err := c.BindJSON(&webhookData)
	if err != nil {
		return err
	}

	for _, alert := range webhookData.Alerts {
		_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
			Namespace: aws.String(config.Namespace),
			MetricData: []*cloudwatch.MetricDatum{
				{
					MetricName: aws.String(alert.Labels["alertname"]),
					Unit:       aws.String("None"),
					Value:      aws.Float64(1),
					Dimensions: getDimensions(&alert.Labels),
				},
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func setupAws() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})

	if err != nil {
		log.Fatal("unable to create AWS session due to an error:", err)
	}

	svc = cloudwatch.New(sess)
}

func setupConfig() {
	var k = koanf.New(".")

	// Default config
	k.Load(confmap.Provider(map[string]interface{}{
		"http-port": 8077,
		"region": "eu-west-1",
		"namespace": "Prometheus",
	}, ""), nil)

	// config.yml
	if err := k.Load(file.Provider("/config/config.yml"), yaml.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	// ENV var config
	k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", "-", -1)
	}), nil)

	k.Unmarshal("", &config)

	log.Printf("config:%+v", config)
}

func main() {
	setupConfig()
	setupAws()

	r := setupRouter()
	listenAddress := fmt.Sprintf(":%d", config.HttpPort)
	log.Printf("listening on: %s", listenAddress)
	if err := r.Run(listenAddress); err != nil {
		panic(err)
	}
}
