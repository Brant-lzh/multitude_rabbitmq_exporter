package main

import (
	"bytes"
	"context"
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	defaultLogLevel = log.InfoLevel
)

func initLogger() {
	log.SetLevel(getLogLevel())
	if strings.ToUpper(config.OutputFormat) == "JSON" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		// The TextFormatter is default, you don't actually have to do this.
		log.SetFormatter(&log.TextFormatter{})
	}
}

func main() {
	//var checkURL = flag.String("check-url", "", "Curl url and return exit code (http: 200 => 0, otherwise 1)")
	var configFile = flag.String("config-file", "conf/rabbitmq.conf", "path to json config")
	flag.Parse()
	log.WithFields(log.Fields{"config-file": *configFile}).Info()
	//if *checkURL != "" { // do a single http get request. Used in docker healthckecks as curl is not inside the image
	//	curl(*checkURL)
	//	return
	//}

	err := initConfigFromFile(*configFile)                  //Try parsing config file
	if _, isPathError := err.(*os.PathError); isPathError { // No file => use environment variables
		initConfig()
	} else if err != nil {
		panic(err)
	}

	initLogger()
	initClient()
	initCacheRegistry()

	//exporter := newExporter()
	//prometheus.MustRegister(exporter)

	log.WithFields(log.Fields{
		"VERSION":    Version,
		"REVISION":   Revision,
		"BRANCH":     Branch,
		"BUILD_DATE": BuildDate,
		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
	}).Info("Starting RabbitMQ exporter")

	log.WithFields(log.Fields{
		"PUBLISH_ADDR": config.PublishAddr,
		"PUBLISH_PORT": config.PublishPort,
		//"RABBIT_URL":          config.RabbitURL,
		"RABBIT_USER":         config.RabbitUsername,
		"RABBIT_CONNECTION":   config.RabbitConnection,
		"OUTPUT_FORMAT":       config.OutputFormat,
		"RABBIT_CAPABILITIES": formatCapabilities(config.RabbitCapabilities),
		"RABBIT_EXPORTERS":    config.EnabledExporters,
		"CAFILE":              config.CAFile,
		"CERTFILE":            config.CertFile,
		"KEYFILE":             config.KeyFile,
		"SKIPVERIFY":          config.InsecureSkipVerify,
		"EXCLUDE_METRICS":     config.ExcludeMetrics,
		"SKIP_EXCHANGES":      config.SkipExchanges.String(),
		"INCLUDE_EXCHANGES":   config.IncludeExchanges.String(),
		"SKIP_QUEUES":         config.SkipQueues.String(),
		"INCLUDE_QUEUES":      config.IncludeQueues.String(),
		"SKIP_VHOST":          config.SkipVHost.String(),
		"INCLUDE_VHOST":       config.IncludeVHost.String(),
		"RABBIT_TIMEOUT":      config.Timeout,
		"MAX_QUEUES":          config.MaxQueues,
		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
	}).Info("Active Configuration")

	handler := http.NewServeMux()
	handler.HandleFunc("/resource/metrics", resourceHandler)
	handler.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	//handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	_, _ = w.Write([]byte(`<html>
	//         <head><title>RabbitMQ Exporter</title></head>
	//         <body>
	//         <h1>RabbitMQ Exporter</h1>
	//         <p><a href='/metrics'>Metrics</a></p>
	//         </body>
	//         </html>`))
	//})
	//handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	//	if exporter.LastScrapeOK() {
	//		w.WriteHeader(http.StatusOK)
	//	} else {
	//		w.WriteHeader(http.StatusGatewayTimeout)
	//	}
	//})
	server := &http.Server{Addr: config.PublishAddr + ":" + config.PublishPort, Handler: handler}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	<-runService()
	log.Info("Shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	cancel()
}

func resourceHandler(w http.ResponseWriter, r *http.Request) {
	req, err := banding(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		unescape, _ := url.QueryUnescape(r.URL.RawQuery)
		_, _ = w.Write([]byte(err.Error() + unescape))
		return
	}
	var (
		re    *multiRegistry
		found bool
		//address string
	)
	if re, found = cache.Get(req.String()); !found {
		//check url
		if !strings.Contains(req.Address, "http") {
			req.Address = "http://" + req.Address
		}

		if valid, _ := regexp.MatchString("https?://[a-zA-Z.0-9]+", strings.ToLower(req.Address)); !valid {
			w.WriteHeader(http.StatusBadRequest)
			unescape, _ := url.QueryUnescape(r.URL.RawQuery)
			_, _ = w.Write([]byte(err.Error() + unescape))
			return
		}

		if err := checkCurl(req.Address); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			unescape, _ := url.QueryUnescape(r.URL.RawQuery)
			_, _ = w.Write([]byte(err.Error() + unescape))
			return
		}

		//get config
		var c = config
		c.RabbitURL = req.Address
		exporter := newExporter(&c)
		re = NewMultiRegistry(exporter)

		//set
		cache.Set(req.String(), re)
	} else {
		re.SetUpdateTime(time.Now())
	}

	promhttp.HandlerFor(re.reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

//func main() {
//	var checkURL = flag.String("check-url", "", "Curl url and return exit code (http: 200 => 0, otherwise 1)")
//	var configFile = flag.String("config-file", "conf/rabbitmq.conf", "path to json config")
//	flag.Parse()
//
//	if *checkURL != "" { // do a single http get request. Used in docker healthckecks as curl is not inside the image
//		curl(*checkURL)
//		return
//	}
//
//	err := initConfigFromFile(*configFile)                  //Try parsing config file
//	if _, isPathError := err.(*os.PathError); isPathError { // No file => use environment variables
//		initConfig()
//	} else if err != nil {
//		panic(err)
//	}
//
//	initLogger()
//	initClient()
//	exporter := newExporter()
//	prometheus.MustRegister(exporter)
//
//	log.WithFields(log.Fields{
//		"VERSION":    Version,
//		"REVISION":   Revision,
//		"BRANCH":     Branch,
//		"BUILD_DATE": BuildDate,
//		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
//	}).Info("Starting RabbitMQ exporter")
//
//	log.WithFields(log.Fields{
//		"PUBLISH_ADDR":        config.PublishAddr,
//		"PUBLISH_PORT":        config.PublishPort,
//		"RABBIT_URL":          config.RabbitURL,
//		"RABBIT_USER":         config.RabbitUsername,
//		"RABBIT_CONNECTION":   config.RabbitConnection,
//		"OUTPUT_FORMAT":       config.OutputFormat,
//		"RABBIT_CAPABILITIES": formatCapabilities(config.RabbitCapabilities),
//		"RABBIT_EXPORTERS":    config.EnabledExporters,
//		"CAFILE":              config.CAFile,
//		"CERTFILE":            config.CertFile,
//		"KEYFILE":             config.KeyFile,
//		"SKIPVERIFY":          config.InsecureSkipVerify,
//		"EXCLUDE_METRICS":     config.ExcludeMetrics,
//		"SKIP_EXCHANGES":      config.SkipExchanges.String(),
//		"INCLUDE_EXCHANGES":   config.IncludeExchanges.String(),
//		"SKIP_QUEUES":         config.SkipQueues.String(),
//		"INCLUDE_QUEUES":      config.IncludeQueues.String(),
//		"SKIP_VHOST":          config.SkipVHost.String(),
//		"INCLUDE_VHOST":       config.IncludeVHost.String(),
//		"RABBIT_TIMEOUT":      config.Timeout,
//		"MAX_QUEUES":          config.MaxQueues,
//		//		"RABBIT_PASSWORD": config.RABBIT_PASSWORD,
//	}).Info("Active Configuration")
//
//	handler := http.NewServeMux()
//	handler.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
//	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		_, _ = w.Write([]byte(`<html>
//            <head><title>RabbitMQ Exporter</title></head>
//            <body>
//            <h1>RabbitMQ Exporter</h1>
//            <p><a href='/metrics'>Metrics</a></p>
//            </body>
//            </html>`))
//	})
//	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
//		if exporter.LastScrapeOK() {
//			w.WriteHeader(http.StatusOK)
//		} else {
//			w.WriteHeader(http.StatusGatewayTimeout)
//		}
//	})
//
//	server := &http.Server{Addr: config.PublishAddr + ":" + config.PublishPort, Handler: handler}
//
//	go func() {
//		if err := server.ListenAndServe(); err != nil {
//			log.Fatal(err)
//		}
//	}()
//
//	<-runService()
//	log.Info("Shutting down")
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	if err := server.Shutdown(ctx); err != nil {
//		log.Fatal(err)
//	}
//	cancel()
//}

func getLogLevel() log.Level {
	lvl := strings.ToLower(os.Getenv("LOG_LEVEL"))
	level, err := log.ParseLevel(lvl)
	if err != nil {
		level = defaultLogLevel
	}
	return level
}

func formatCapabilities(caps rabbitCapabilitySet) string {
	var buffer bytes.Buffer
	first := true
	for k := range caps {
		if !first {
			buffer.WriteString(",")
		}
		first = false
		buffer.WriteString(string(k))
	}
	return buffer.String()
}
