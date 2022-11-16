package kube

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	log "github.com/sirupsen/logrus"
)

const (
	kubeconfigTemplateStr = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{ .CAData }}
    server: {{ .ServerURL }}
  name: {{ .Name }}
contexts:
- context:
    cluster: {{ .Name }}
    user: {{ .Name }}
  name: {{ .Name }}
current-context: {{ .Name }}
kind: Config
preferences: {}
users:
- name: {{ .Name }}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - token
      - --region
      - {{ .Region }}
      - --cluster-id
      - {{ .Name }}
      command: aws-iam-authenticator`
)

var (
	kubeconfigTemplate = template.Must(
		template.New("kubeconfig").Parse(kubeconfigTemplateStr),
	)
)

// KubeconfigTemplateData stores the data necessary to generate a kubeconfig.
type KubeconfigTemplateData struct {
	Name      string
	ServerURL string
	CAData    string
	Region    string
}

// CreateKubeconfigFromClusterData generates a kubeconfig from the raw components in
// KubeconfigTemplateData.
func CreateKubeconfigFromClusterData(
	clusterName string,
	serverURL string,
	caData string,
	region string,
	path string,
) error {
	log.Infof("Creating kubeconfig for cluster %s in %s", clusterName, path)

	data := KubeconfigTemplateData{
		Name:      clusterName,
		ServerURL: serverURL,
		CAData:    caData,
		Region:    region,
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	return kubeconfigTemplate.Execute(out, data)
}

// CreateKubeconfigViaAPI generates a kubeconfig by hitting the EKS API.
func CreateKubeconfigViaAPI(
	ctx context.Context,
	sess *session.Session,
	clusterName string,
	region string,
	path string,
) error {
	eksClient := eks.New(sess)
	resp, err := eksClient.DescribeClusterWithContext(
		ctx,
		&eks.DescribeClusterInput{
			Name: aws.String(clusterName),
		},
	)
	if err != nil {
		return err
	}
	cluster := resp.Cluster

	return CreateKubeconfigFromClusterData(
		clusterName,
		aws.StringValue(cluster.Endpoint),
		aws.StringValue(cluster.CertificateAuthority.Data),
		region,
		path,
	)
}

// KubeconfigMatchesCluster determines (roughly) whether a kubeconfig matches the provided
// cluster name. Currently, it just looks for the latter string in the config.
func KubeconfigMatchesCluster(path string, clusterName string) bool {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("Error reading kubeconfig from %s: %+v", path, err)
		return false
	}

	return bytes.Contains(contents, []byte(clusterName))
}
