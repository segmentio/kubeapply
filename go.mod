module github.com/segmentio/kubeapply

go 1.15

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 v3.2.1
	github.com/aws/aws-lambda-go v1.15.0
	github.com/aws/aws-sdk-go v1.29.16
	github.com/briandowns/spinner v1.11.1
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.1
	github.com/google/go-github/v30 v30.0.0
	github.com/gorilla/mux v1.7.4
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.11 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/pmezard/go-difflib v1.0.0
	github.com/segmentio/conf v1.2.0
	github.com/segmentio/encoding v0.2.7
	github.com/segmentio/stats v3.0.0+incompatible
	github.com/segmentio/stats/v4 v4.5.3
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/stripe/skycfg v0.0.0-20200303020846-4f599970a3e6
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/yannh/kubeconform v0.4.6
	github.com/zorkian/go-datadog-api v2.28.0+incompatible // indirect
	go.starlark.net v0.0.0-20201204201740-42d4f566359b
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/validator.v2 v2.0.0-20180514200540-135c24b11c19 // indirect
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200601152816-913338de1bd2 // indirect
	gopkg.in/zorkian/go-datadog-api.v2 v2.28.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog/v2 v2.4.0
	k8s.io/kubectl v0.20.2
)

// Need to pin to older version to get around https://github.com/stripe/skycfg/issues/86.
replace github.com/golang/protobuf v1.4.3 => github.com/golang/protobuf v1.3.2
