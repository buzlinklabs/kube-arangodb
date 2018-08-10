//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/pflag"
)

var (
	options struct {
		OutputSuffix string
		TemplatesDir string

		Namespace                         string
		Image                             string
		ImagePullPolicy                   string
		ImageSHA256                       bool
		DeploymentOperatorName            string
		DeploymentReplicationOperatorName string
		StorageOperatorName               string
		RBAC                              bool
		AllowChaos                        bool
	}
	deploymentTemplateNames = []Template{
		Template{Name: "rbac.yaml", Predicate: hasRBAC},
		Template{Name: "deployment.yaml"},
		Template{Name: "service.yaml"},
	}
	deploymentReplicationTemplateNames = []Template{
		Template{Name: "rbac.yaml", Predicate: hasRBAC},
		Template{Name: "deployment-replication.yaml"},
		Template{Name: "service.yaml"},
	}
	storageTemplateNames = []Template{
		Template{Name: "rbac.yaml", Predicate: hasRBAC},
		Template{Name: "deployment.yaml"},
		Template{Name: "service.yaml"},
	}
	testTemplateNames = []Template{
		Template{Name: "rbac.yaml", Predicate: func(o TemplateOptions, isHelm bool) bool { return o.RBAC && !isHelm }},
	}
)

type Template struct {
	Name      string
	Predicate func(o TemplateOptions, isHelm bool) bool
}

type TemplateGroup struct {
	ChartName string
	Templates []Template
}

func hasRBAC(o TemplateOptions, isHelm bool) bool {
	return o.RBAC || isHelm
}

var (
	tmplFuncs = template.FuncMap{
		"quote": func(x string) string { return strconv.Quote(x) },
	}
)

type (
	chartTemplates map[string]string
)

const (
	kubeArangoDBChartTemplate = `
apiVersion: v1
name: kube-arangodb
version: "{{ .Version }}"
description: |
  Kube-ArangoDB is a set of operators to easily deploy ArangoDB deployments on Kubernetes
home: https://arangodb.com
`
	kubeArangoDBStorageChartTemplate = `
apiVersion: v1
name: kube-arangodb-storage
version: "{{ .Version }}"
description: |
  Kube-ArangoDB-Storage is a cluster-wide operator used to provision PersistentVolumes on disks attached locally to Nodes
home: https://arangodb.com
`
	kubeArangoDBValuesTemplate = `
Image: {{ .Image | quote }}
ImagePullPolicy: {{ .ImagePullPolicy | quote }}
RBAC:
  Create: {{ .RBAC }}
Deployment:
  Create: {{ .Deployment.Create }}
  User:
    ServiceAccountName: {{ .Deployment.User.ServiceAccountName | quote }}
  Operator:
    ServiceAccountName: {{ .Deployment.Operator.ServiceAccountName | quote }}
    ServiceType: {{ .Deployment.Operator.ServiceType | quote }}
  AllowChaos: {{ .Deployment.AllowChaos }}
DeploymentReplication:
  Create: {{ .DeploymentReplication.Create }}
  User:
    ServiceAccountName: {{ .DeploymentReplication.User.ServiceAccountName | quote }}
  Operator:
    ServiceAccountName: {{ .DeploymentReplication.Operator.ServiceAccountName | quote }}
    ServiceType: {{ .DeploymentReplication.Operator.ServiceType | quote }}
`
	kubeArangoDBStorageValuesTemplate = `
Image: {{ .Image | quote }}
ImagePullPolicy: {{ .ImagePullPolicy | quote }}
RBAC:
  Create: {{ .RBAC }}
Storage:
  User:
    ServiceAccountName: {{ .Storage.User.ServiceAccountName | quote }}
  Operator:
    ServiceAccountName: {{ .Storage.Operator.ServiceAccountName | quote }}
    ServiceType: {{ .Storage.Operator.ServiceType | quote }}
`
)

func init() {
	pflag.StringVar(&options.OutputSuffix, "output-suffix", "", "Suffix of the generated manifest files")
	pflag.StringVar(&options.TemplatesDir, "templates-dir", "manifests/templates", "Directory containing manifest templates")
	pflag.StringVar(&options.Namespace, "namespace", "default", "Namespace in which the operator will be deployed")
	pflag.StringVar(&options.Image, "image", "arangodb/arangodb-operator:latest", "Fully qualified image name of the ArangoDB operator")
	pflag.StringVar(&options.ImagePullPolicy, "image-pull-policy", "IfNotPresent", "Pull policy of the ArangoDB operator image")
	pflag.BoolVar(&options.ImageSHA256, "image-sha256", true, "Use SHA256 syntax for image")
	pflag.StringVar(&options.DeploymentOperatorName, "deployment-operator-name", "arango-deployment-operator", "Name of the ArangoDeployment operator deployment")
	pflag.StringVar(&options.DeploymentReplicationOperatorName, "deployment-replication-operator-name", "arango-deployment-replication-operator", "Name of the ArangoDeploymentReplication operator deployment")
	pflag.StringVar(&options.StorageOperatorName, "storage-operator-name", "arango-storage-operator", "Name of the ArangoLocalStorage operator deployment")
	pflag.BoolVar(&options.RBAC, "rbac", true, "Use role based access control")
	pflag.BoolVar(&options.AllowChaos, "allow-chaos", false, "If set, allows chaos in deployments")

	pflag.Parse()
}

type TemplateOptions struct {
	Version               string
	Image                 string
	ImagePullPolicy       string
	RBAC                  bool
	RBACFilterStart       string
	RBACFilterEnd         string
	Deployment            ResourceOptions
	DeploymentReplication ResourceOptions
	Storage               ResourceOptions
	Test                  CommonOptions
}

type CommonOptions struct {
	Namespace          string
	RoleName           string
	RoleBindingName    string
	ServiceAccountName string
	ServiceType        string
}

type ResourceOptions struct {
	Create                 string
	FilterStart            string
	FilterEnd              string
	User                   CommonOptions
	Operator               CommonOptions
	OperatorDeploymentName string
	AllowChaos             string
}

func main() {
	// Check options
	if options.Namespace == "" {
		log.Fatal("--namespace not specified.")
	}
	if options.Image == "" {
		log.Fatal("--image not specified.")
	}

	// Fetch image sha256
	if options.ImageSHA256 {
		cmd := exec.Command(
			"docker",
			"inspect",
			"--format={{index .RepoDigests 0}}",
			options.Image,
		)
		result, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(string(result))
			log.Fatalf("Failed to fetch image SHA256: %v", err)
		}
		options.Image = strings.TrimSpace(string(result))
	}

	// Prepare templates to include
	templateInfoSet := map[string]TemplateGroup{
		"deployment":             TemplateGroup{ChartName: "kube-arangodb", Templates: deploymentTemplateNames},
		"deployment-replication": TemplateGroup{ChartName: "kube-arangodb", Templates: deploymentReplicationTemplateNames},
		"storage":                TemplateGroup{ChartName: "kube-arangodb-storage", Templates: storageTemplateNames},
		"test":                   TemplateGroup{ChartName: "", Templates: testTemplateNames},
	}

	// Read VERSION
	version, err := ioutil.ReadFile("VERSION")
	if err != nil {
		log.Fatalf("Failed to read VERSION file: %v", err)
	}

	// Process templates
	templateOptions := TemplateOptions{
		Version:         strings.TrimSpace(string(version)),
		Image:           options.Image,
		ImagePullPolicy: options.ImagePullPolicy,
		RBAC:            options.RBAC,
		Deployment: ResourceOptions{
			Create: "true",
			User: CommonOptions{
				Namespace:          options.Namespace,
				RoleName:           "arango-deployments",
				RoleBindingName:    "arango-deployments",
				ServiceAccountName: "default",
			},
			Operator: CommonOptions{
				Namespace:          options.Namespace,
				RoleName:           "arango-deployment-operator",
				RoleBindingName:    "arango-deployment-operator",
				ServiceAccountName: "default",
				ServiceType:        "ClusterIP",
			},
			OperatorDeploymentName: "arango-deployment-operator",
			AllowChaos:             strconv.FormatBool(options.AllowChaos),
		},
		DeploymentReplication: ResourceOptions{
			Create: "true",
			User: CommonOptions{
				Namespace:          options.Namespace,
				RoleName:           "arango-deployment-replications",
				RoleBindingName:    "arango-deployment-replications",
				ServiceAccountName: "default",
			},
			Operator: CommonOptions{
				Namespace:          options.Namespace,
				RoleName:           "arango-deployment-replication-operator",
				RoleBindingName:    "arango-deployment-replication-operator",
				ServiceAccountName: "default",
				ServiceType:        "ClusterIP",
			},
			OperatorDeploymentName: "arango-deployment-replication-operator",
		},
		Storage: ResourceOptions{
			Create: "true",
			User: CommonOptions{
				Namespace:          options.Namespace,
				RoleName:           "arango-storages",
				RoleBindingName:    "arango-storages",
				ServiceAccountName: "default",
			},
			Operator: CommonOptions{
				Namespace:          "kube-system",
				RoleName:           "arango-storage-operator",
				RoleBindingName:    "arango-storage-operator",
				ServiceAccountName: "arango-storage-operator",
				ServiceType:        "ClusterIP",
			},
			OperatorDeploymentName: "arango-storage-operator",
		},
		Test: CommonOptions{
			Namespace:          options.Namespace,
			RoleName:           "arango-operator-test",
			RoleBindingName:    "arango-operator-test",
			ServiceAccountName: "default",
		},
	}
	chartTemplateOptions := TemplateOptions{
		Version:         strings.TrimSpace(string(version)),
		RBACFilterStart: "{{- if .Values.RBAC.Create }}",
		RBACFilterEnd:   "{{- end }}",
		Image:           "{{ .Values.Image }}",
		ImagePullPolicy: "{{ .Values.ImagePullPolicy }}",
		Deployment: ResourceOptions{
			Create:      "{{ .Values.Deployment.Create }}",
			FilterStart: "{{- if .Values.Deployment.Create }}",
			FilterEnd:   "{{- end }}",
			User: CommonOptions{
				Namespace:          "{{ .Release.Namespace }}",
				RoleName:           `{{ printf "%s-%s" .Release.Name "deployments" | trunc 63 | trimSuffix "-" }}`,
				RoleBindingName:    `{{ printf "%s-%s" .Release.Name "deployments" | trunc 63 | trimSuffix "-" }}`,
				ServiceAccountName: "{{ .Values.Deployment.User.ServiceAccountName }}",
			},
			Operator: CommonOptions{
				Namespace:          "{{ .Release.Namespace }}",
				RoleName:           `{{ printf "%s-%s" .Release.Name "deployment-operator" | trunc 63 | trimSuffix "-" }}`,
				RoleBindingName:    `{{ printf "%s-%s" .Release.Name "deployment-operator" | trunc 63 | trimSuffix "-" }}`,
				ServiceAccountName: "{{ .Values.Deployment.Operator.ServiceAccountName }}",
				ServiceType:        "{{ .Values.Deployment.Operator.ServiceType }}",
			},
			OperatorDeploymentName: `{{ printf "%s-%s" .Release.Name "deployment-operator" | trunc 63 | trimSuffix "-" }}`,
			AllowChaos:             "{{ .Values.Deployment.AllowChaos }}",
		},
		DeploymentReplication: ResourceOptions{
			Create:      "{{ .Values.DeploymentReplication.Create }}",
			FilterStart: "{{- if .Values.DeploymentReplication.Create }}",
			FilterEnd:   "{{- end }}",
			User: CommonOptions{
				Namespace:          "{{ .Release.Namespace }}",
				RoleName:           `{{ printf "%s-%s" .Release.Name "deployment-replications" | trunc 63 | trimSuffix "-" }}`,
				RoleBindingName:    `{{ printf "%s-%s" .Release.Name "deployment-replications" | trunc 63 | trimSuffix "-" }}`,
				ServiceAccountName: "{{ .Values.DeploymentReplication.User.ServiceAccountName }}",
			},
			Operator: CommonOptions{
				Namespace:          "{{ .Release.Namespace }}",
				RoleName:           `{{ printf "%s-%s" .Release.Name "deployment-replication-operator" | trunc 63 | trimSuffix "-" }}`,
				RoleBindingName:    `{{ printf "%s-%s" .Release.Name "deployment-replication-operator" | trunc 63 | trimSuffix "-" }}`,
				ServiceAccountName: "{{ .Values.DeploymentReplication.Operator.ServiceAccountName }}",
				ServiceType:        "{{ .Values.DeploymentReplication.Operator.ServiceType }}",
			},
			OperatorDeploymentName: `{{ printf "%s-%s" .Release.Name "deployment-replication-operator" | trunc 63 | trimSuffix "-" }}`,
		},
		Storage: ResourceOptions{
			Create: "{{ .Values.Storage.Create }}",
			User: CommonOptions{
				Namespace:          "{{ .Release.Namespace }}",
				RoleName:           `{{ printf "%s-%s" .Release.Name "storages" | trunc 63 | trimSuffix "-" }}`,
				RoleBindingName:    `{{ printf "%s-%s" .Release.Name "storages" | trunc 63 | trimSuffix "-" }}`,
				ServiceAccountName: "{{ .Values.Storage.User.ServiceAccountName }}",
			},
			Operator: CommonOptions{
				Namespace:          "kube-system",
				RoleName:           `{{ printf "%s-%s" .Release.Name "storage-operator" | trunc 63 | trimSuffix "-" }}`,
				RoleBindingName:    `{{ printf "%s-%s" .Release.Name "storage-operator" | trunc 63 | trimSuffix "-" }}`,
				ServiceAccountName: "{{ .Values.Storage.Operator.ServiceAccountName }}",
				ServiceType:        "{{ .Values.Storage.Operator.ServiceType }}",
			},
			OperatorDeploymentName: `{{ printf "%s-%s" .Release.Name "storage-operator" | trunc 63 | trimSuffix "-" }}`,
		},
	}

	for group, templateGroup := range templateInfoSet {
		// Build standalone yaml file for this group
		{
			output := &bytes.Buffer{}
			for i, tempInfo := range templateGroup.Templates {
				if tempInfo.Predicate == nil || tempInfo.Predicate(templateOptions, false) {
					name := tempInfo.Name
					t, err := template.New(name).ParseFiles(filepath.Join(options.TemplatesDir, group, name))
					if err != nil {
						log.Fatalf("Failed to parse template %s: %v", name, err)
					}
					if i > 0 {
						output.WriteString("\n---\n\n")
					}
					output.WriteString(fmt.Sprintf("## %s/%s\n", group, name))
					t.Execute(output, templateOptions)
					output.WriteString("\n")
				}
			}

			// Save output
			if output.Len() > 0 {
				outputDir, err := filepath.Abs("manifests")
				if err != nil {
					log.Fatalf("Failed to get absolute output dir: %v\n", err)
				}
				outputPath := filepath.Join(outputDir, "arango-"+group+options.OutputSuffix+".yaml")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					log.Fatalf("Failed to create output directory: %v\n", err)
				}
				if err := ioutil.WriteFile(outputPath, output.Bytes(), 0644); err != nil {
					log.Fatalf("Failed to write output file: %v\n", err)
				}
			}
		}

		// Build helm template file for this group
		{
			output := &bytes.Buffer{}
			for i, tempInfo := range templateGroup.Templates {
				if tempInfo.Predicate == nil || tempInfo.Predicate(chartTemplateOptions, true) {
					name := tempInfo.Name
					t, err := template.New(name).ParseFiles(filepath.Join(options.TemplatesDir, group, name))
					if err != nil {
						log.Fatalf("Failed to parse template %s: %v", name, err)
					}
					if i > 0 {
						output.WriteString("\n---\n\n")
					}
					output.WriteString(fmt.Sprintf("## %s/%s\n", group, name))
					t.Execute(output, chartTemplateOptions)
					output.WriteString("\n")
				}
			}

			// Save output
			if output.Len() > 0 {
				outputDir, err := filepath.Abs(filepath.Join("bin", "charts", templateGroup.ChartName, "templates"))
				if err != nil {
					log.Fatalf("Failed to get absolute output dir: %v\n", err)
				}
				outputPath := filepath.Join(outputDir, group+".yaml")
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					log.Fatalf("Failed to create output directory: %v\n", err)
				}
				if err := ioutil.WriteFile(outputPath, output.Bytes(), 0644); err != nil {
					log.Fatalf("Failed to write output file: %v\n", err)
				}
			}
		}
	}

	// Build Chart files
	chartTemplateGroups := map[string]chartTemplates{
		"kube-arangodb": chartTemplates{
			"Chart.yaml":  kubeArangoDBChartTemplate,
			"values.yaml": kubeArangoDBValuesTemplate,
		},
		"kube-arangodb-storage": chartTemplates{
			"Chart.yaml":  kubeArangoDBStorageChartTemplate,
			"values.yaml": kubeArangoDBStorageValuesTemplate,
		},
	}
	for groupName, chartTemplates := range chartTemplateGroups {
		for name, templateSource := range chartTemplates {
			output := &bytes.Buffer{}
			t, err := template.New(name).Funcs(tmplFuncs).Parse(templateSource)
			if err != nil {
				log.Fatalf("Failed to parse template %s: %v", name, err)
			}
			t.Execute(output, templateOptions)

			// Save output
			outputDir, err := filepath.Abs(filepath.Join("bin", "charts", groupName))
			if err != nil {
				log.Fatalf("Failed to get absolute output dir: %v\n", err)
			}
			outputPath := filepath.Join(outputDir, name)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				log.Fatalf("Failed to create output directory: %v\n", err)
			}
			if err := ioutil.WriteFile(outputPath, output.Bytes(), 0644); err != nil {
				log.Fatalf("Failed to write output file: %v\n", err)
			}
		}
	}
}
