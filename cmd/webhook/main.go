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
	"github.com/spf13/viper"
)

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
	Labels       Labels
	Annotations  Annotations
	StartsAt     string
	EndsAt       string
	GeneratorURL string
}

// Labels - Contains the labels for an alert
type Labels struct {
	Name                 string `json:"__name__"`
	Alertname            string
	App                  string
	Backend              string
	Instance             string
	Job                  string
	KuberenetesNamespace string `json:"kubernetes_namespace"`
	KubernetesPodName    string `json:"kubernetes_pod_name"`
	PodTemplateHash      string `json:"pod_template_hash"`
	TrafficType          string `json:"traffic_type"`
	URL                  string
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
var webhookData Webhook

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
	r.POST("/webhook", webhook)

	return r
}

func putMetric(c *gin.Context) error {
	err := c.BindJSON(&webhookData)
	if err != nil {
		return err
	}

	svc = cloudwatch.New(sess)
	for _, alert := range webhookData.Alerts {
		_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
			Namespace: aws.String(viper.GetString("namespace")),
			MetricData: []*cloudwatch.MetricDatum{
				{
					MetricName: aws.String(alert.Labels.Alertname),
					Unit:       aws.String("None"),
					Value:      aws.Float64(1),
					Dimensions: []*cloudwatch.Dimension{},
				},
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("http-port", 8077)
	viper.SetDefault("region", "eu-west-1")
	viper.SetDefault("namespace", "Prometheus")

	var err error

	sess, err = session.NewSession(&aws.Config{Region: aws.String(viper.GetString("region"))})
	if err != nil {
		log.Fatal("unable to create AWS session due to an error:", err)
	}

	r := setupRouter()
	listenAddress := fmt.Sprintf(":%d", viper.GetInt("http-port"))
	log.Printf("listening on: %s", listenAddress)
	if err := r.Run(listenAddress); err != nil {
		panic(err)
	}
}
