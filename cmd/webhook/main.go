package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"strings"
)

func webhook(c *gin.Context) {
	if err := putMetric(); err != nil {
		log.Println("unable to put metric: ", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, struct{}{})
}

var sess *session.Session
var svc *cloudwatch.CloudWatch

func main() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("http-port", 8077)
	viper.SetDefault("metric-name", "DeadMansSwitch")
	viper.SetDefault("region", "eu-west-1")

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

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, struct{}{})
	})
	r.POST("/webhook", webhook)

	return r
}

func putMetric() error {
	svc = cloudwatch.New(sess)

	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Prometheus"),
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String(viper.GetString("metric-name")),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(1),
				Dimensions: []*cloudwatch.Dimension{
					/*
						{
							Name:  aws.String("InstanceName"),
							Value: aws.String("prometheus-k8s-0"),
						},
					*/
				},
			},
		},
	})

	return err
}
