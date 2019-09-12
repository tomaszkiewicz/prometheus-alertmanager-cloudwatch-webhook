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

var sess *session.Session
var svc *cloudwatch.CloudWatch

func webhook(c *gin.Context) {
	if err := putMetric(); err != nil {
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

func putMetric() error {
	svc = cloudwatch.New(sess)

	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String(viper.GetString("NAMESPACE")),
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String(viper.GetString("METRIC_NAME")),
				Unit:       aws.String("None"),
				Value:      aws.Float64(1),
				Dimensions: []*cloudwatch.Dimension{},
			},
		},
	})

	return err
}

func main() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("HTTP_PORT", 8077)
	viper.SetDefault("METRIC_NAME", "DeadMansSwitch")
	viper.SetDefault("REGION", "eu-west-1")
	viper.SetDefault("NAMESPACE", "Prometheus")

	var err error

	sess, err = session.NewSession(&aws.Config{Region: aws.String(viper.GetString("REGION"))})
	if err != nil {
		log.Fatal("unable to create AWS session due to an error:", err)
	}

	r := setupRouter()
	listenAddress := fmt.Sprintf(":%d", viper.GetInt("HTTP_PORT"))
	log.Printf("listening on: %s", listenAddress)
	if err := r.Run(listenAddress); err != nil {
		panic(err)
	}
}
