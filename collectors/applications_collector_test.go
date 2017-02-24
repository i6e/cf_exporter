package collectors_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/prometheus/client_golang/prometheus"

	. "github.com/cloudfoundry-community/cf_exporter/collectors"
	. "github.com/cloudfoundry-community/cf_exporter/utils/test_matchers"
)

var _ = Describe("ApplicationsCollectors", func() {
	var (
		err      error
		server   *ghttp.Server
		cfClient *cfclient.Client

		applicationInfoMetric                       *prometheus.GaugeVec
		applicationInstancesMetric                  *prometheus.GaugeVec
		applicationMemoryMbMetric                   *prometheus.GaugeVec
		applicationDiskQuotaMbMetric                *prometheus.GaugeVec
		applicationsScrapesTotalMetric              prometheus.Counter
		applicationsScrapeErrorsTotalMetric         prometheus.Counter
		lastApplicationsScrapeErrorMetric           prometheus.Gauge
		lastApplicationsScrapeTimestampMetric       prometheus.Gauge
		lastApplicationsScrapeDurationSecondsMetric prometheus.Gauge

		namespace   = "test_namespace"
		environment = "test_environment"
		deployment  = "test_deployment"

		applicationId1    = "fake_application_id_1"
		applicationName1  = "fake_application_name_1"
		buildpack1        = "fake_buildpack_1"
		organizationId1   = "fake_organization_id_1"
		organizationName1 = "fake_organization_name_1"
		spaceId1          = "fake_space_id_1"
		spaceName1        = "fake_space_name_1"
		stackId1          = "fake_stack_id_1"
		state1            = "fake_state_1"
		instances1        = 11
		memoryMb1         = 21
		diskQuotaMb1      = 31

		applicationId2    = "fake_application_id_2"
		applicationName2  = "fake_application_name_2"
		buildpack2        = "fake_buildpack_2"
		organizationId2   = "fake_organization_id_2"
		organizationName2 = "fake_organization_name_2"
		spaceId2          = "fake_space_id_2"
		spaceName2        = "fake_space_name_2"
		stackId2          = "fake_stack_id_2"
		state2            = "fake_state_2"
		instances2        = 12
		memoryMb2         = 22
		diskQuotaMb2      = 32

		applicationsCollector *ApplicationsCollector
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/info"),
				ghttp.RespondWith(http.StatusOK, "{}"),
			),
		)

		cfClient, err = cfclient.NewClient(&cfclient.Config{
			ApiAddress: server.URL(),
			Token:      "fake-token",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(server.ReceivedRequests())).To(Equal(1))

		applicationInfoMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "application",
				Name:        "info",
				Help:        "Labeled Cloud Foundry Application information with a constant '1' value.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
			[]string{"application_id", "application_name", "buildpack", "organization_id", "organization_name", "space_id", "space_name", "stack_id", "state"},
		)
		applicationInfoMetric.WithLabelValues(applicationId1, applicationName1, buildpack1, organizationId1, organizationName1, spaceId1, spaceName1, stackId1, state1).Set(1)
		applicationInfoMetric.WithLabelValues(applicationId2, applicationName2, buildpack2, organizationId2, organizationName2, spaceId2, spaceName2, stackId2, state2).Set(1)

		applicationInstancesMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "application",
				Name:        "instances",
				Help:        "Cloud Foundry Application Instances.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
			[]string{"application_id", "application_name", "organization_id", "organization_name", "space_id", "space_name"},
		)
		applicationInstancesMetric.WithLabelValues(applicationId1, applicationName1, organizationId1, organizationName1, spaceId1, spaceName1).Set(float64(instances1))
		applicationInstancesMetric.WithLabelValues(applicationId2, applicationName2, organizationId2, organizationName2, spaceId2, spaceName2).Set(float64(instances2))

		applicationMemoryMbMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "application",
				Name:        "memory_mb",
				Help:        "Cloud Foundry Application Memory (Mb).",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
			[]string{"application_id", "application_name", "organization_id", "organization_name", "space_id", "space_name"},
		)
		applicationMemoryMbMetric.WithLabelValues(applicationId1, applicationName1, organizationId1, organizationName1, spaceId1, spaceName1).Set(float64(memoryMb1))
		applicationMemoryMbMetric.WithLabelValues(applicationId2, applicationName2, organizationId2, organizationName2, spaceId2, spaceName2).Set(float64(memoryMb2))

		applicationDiskQuotaMbMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "application",
				Name:        "disk_quota_mb",
				Help:        "Cloud Foundry Application Disk Quota (Mb).",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
			[]string{"application_id", "application_name", "organization_id", "organization_name", "space_id", "space_name"},
		)
		applicationDiskQuotaMbMetric.WithLabelValues(applicationId1, applicationName1, organizationId1, organizationName1, spaceId1, spaceName1).Set(float64(diskQuotaMb1))
		applicationDiskQuotaMbMetric.WithLabelValues(applicationId2, applicationName2, organizationId2, organizationName2, spaceId2, spaceName2).Set(float64(diskQuotaMb2))

		applicationsScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "applications_scrapes",
				Name:        "total",
				Help:        "Total number of scrapes for Cloud Foundry Applications.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)
		applicationsScrapesTotalMetric.Inc()

		applicationsScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "applications_scrape_errors",
				Name:        "total",
				Help:        "Total number of scrape errors of Cloud Foundry Applications.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)

		lastApplicationsScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_applications_scrape_error",
				Help:        "Whether the last scrape of Applications metrics from Cloud Foundry resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)

		lastApplicationsScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_applications_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Applications metrics from Cloud Foundry.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)

		lastApplicationsScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_applications_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Applications metrics from Cloud Foundry.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)
	})

	JustBeforeEach(func() {
		applicationsCollector = NewApplicationsCollector(namespace, environment, deployment, cfClient)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Describe", func() {
		var (
			descriptions chan *prometheus.Desc
		)

		BeforeEach(func() {
			descriptions = make(chan *prometheus.Desc)
		})

		JustBeforeEach(func() {
			go applicationsCollector.Describe(descriptions)
		})

		It("returns a application_info metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(applicationInfoMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				buildpack1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
				stackId1,
				state1,
			).Desc())))
		})

		It("returns a application_instances metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(applicationInstancesMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
			).Desc())))
		})

		It("returns a application_memory_mb metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(applicationMemoryMbMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
			).Desc())))
		})

		It("returns a application_disk_quota_mb metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(applicationDiskQuotaMbMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
			).Desc())))
		})

		It("returns a applications_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(applicationsScrapesTotalMetric.Desc())))
		})

		It("returns a applications_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(applicationsScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_applications_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastApplicationsScrapeErrorMetric.Desc())))
		})

		It("returns a last_applications_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastApplicationsScrapeTimestampMetric.Desc())))
		})

		It("returns a last_applications_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastApplicationsScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode     int
			orgsResponse   cfclient.OrgResponse
			spacesResponse cfclient.SpaceResponse
			appsResponse   cfclient.AppResponse
			metrics        chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK

			orgsResponse = cfclient.OrgResponse{
				Resources: []cfclient.OrgResource{
					cfclient.OrgResource{
						Meta: cfclient.Meta{
							Guid: organizationId1,
						},
						Entity: cfclient.Org{
							Name: organizationName1,
						},
					},
					cfclient.OrgResource{
						Meta: cfclient.Meta{
							Guid: organizationId2,
						},
						Entity: cfclient.Org{
							Name: organizationName2,
						},
					},
				},
			}

			spacesResponse = cfclient.SpaceResponse{
				Resources: []cfclient.SpaceResource{
					cfclient.SpaceResource{
						Meta: cfclient.Meta{
							Guid: spaceId1,
						},
						Entity: cfclient.Space{
							Name:             spaceName1,
							OrganizationGuid: organizationId1,
						},
					},
					cfclient.SpaceResource{
						Meta: cfclient.Meta{
							Guid: spaceId2,
						},
						Entity: cfclient.Space{
							Name:             spaceName2,
							OrganizationGuid: organizationId2,
						},
					},
				},
			}

			appsResponse = cfclient.AppResponse{
				Resources: []cfclient.AppResource{
					cfclient.AppResource{
						Meta: cfclient.Meta{
							Guid: applicationId1,
						},
						Entity: cfclient.App{
							Name:      applicationName1,
							Memory:    memoryMb1,
							Instances: instances1,
							DiskQuota: diskQuotaMb1,
							SpaceGuid: spaceId1,
							StackGuid: stackId1,
							State:     state1,
							Buildpack: buildpack1,
						},
					},
					cfclient.AppResource{
						Meta: cfclient.Meta{
							Guid: applicationId2,
						},
						Entity: cfclient.App{
							Name:      applicationName2,
							Memory:    memoryMb2,
							Instances: instances2,
							DiskQuota: diskQuotaMb2,
							SpaceGuid: spaceId2,
							StackGuid: stackId2,
							State:     state2,
							Buildpack: buildpack2,
						},
					},
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v2/organizations"),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &orgsResponse),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v2/spaces"),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &spacesResponse),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v2/apps"),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &appsResponse),
				),
			)

			go applicationsCollector.Collect(metrics)
		})

		It("returns an application_info metric for application 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationInfoMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				buildpack1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
				stackId1,
				state1,
			))))
		})

		It("returns an application_info metric for application 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationInfoMetric.WithLabelValues(
				applicationId2,
				applicationName2,
				buildpack2,
				organizationId2,
				organizationName2,
				spaceId2,
				spaceName2,
				stackId2,
				state2,
			))))
		})

		It("returns an application_instances metric for application 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationInstancesMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
			))))
		})

		It("returns an application_instances metric for application 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationInstancesMetric.WithLabelValues(
				applicationId2,
				applicationName2,
				organizationId2,
				organizationName2,
				spaceId2,
				spaceName2,
			))))
		})

		It("returns an application_memory_mb metric for application 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationMemoryMbMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
			))))
		})

		It("returns an application_memory_mb metric for application 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationMemoryMbMetric.WithLabelValues(
				applicationId2,
				applicationName2,
				organizationId2,
				organizationName2,
				spaceId2,
				spaceName2,
			))))
		})

		It("returns an application_disk_quota_mb metric for application 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationDiskQuotaMbMetric.WithLabelValues(
				applicationId1,
				applicationName1,
				organizationId1,
				organizationName1,
				spaceId1,
				spaceName1,
			))))
		})

		It("returns an application_disk_quota_mb metric for application 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationDiskQuotaMbMetric.WithLabelValues(
				applicationId2,
				applicationName2,
				organizationId2,
				organizationName2,
				spaceId2,
				spaceName2,
			))))
		})

		It("returns an applications_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationsScrapesTotalMetric)))
		})

		It("returns an applications_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(applicationsScrapeErrorsTotalMetric)))
		})

		It("returns a last_applications_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastApplicationsScrapeErrorMetric)))
		})

		Context("when it fails to list the applications", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError

				//TODO: go-cfclient does not yet check for the http status code
				//applicationsScrapeErrorsTotalMetric.Inc()
				//lastApplicationsScrapeErrorMetric.Set(1)
			})

			It("returns an applications_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(applicationsScrapeErrorsTotalMetric)))
			})

			It("returns a last_applications_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastApplicationsScrapeErrorMetric)))
			})
		})
	})
})
