package oauth

import (
	"github.com/fusor/cpma/pkg/transform/configmaps"
	"github.com/fusor/cpma/pkg/transform/secrets"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"

	configv1 "github.com/openshift/api/legacyconfig/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
)

func init() {
	oauthv1.Install(scheme.Scheme)
	configv1.InstallLegacy(scheme.Scheme)
}

// reference:
//   [v3] OCPv3:
//   - [1] https://docs.openshift.com/container-platform/3.11/install_config/configuring_authentication.html#identity_providers_master_config
//   [v4] OCPv4:
//   - [2] htpasswd: https://docs.openshift.com/container-platform/4.0/authentication/understanding-identity-provider.html
//   - [3] github: https://docs.openshift.com/container-platform/4.0/authentication/identity_providers/configuring-github-identity-provider.html

// CRD Shared CRD part, present in all types of OAuth CRDs
type CRD struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   MetaData `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Spec is a CRD Spec
type Spec struct {
	IdentityProviders []interface{} `yaml:"identityProviders"`
}

type identityProviderCommon struct {
	Name          string `yaml:"name"`
	Challenge     bool   `yaml:"challenge"`
	Login         bool   `yaml:"login"`
	MappingMethod string `yaml:"mappingMethod"`
	Type          string `yaml:"type"`
}

// MetaData contains CRD Metadata
type MetaData struct {
	Name      string `yaml:"name"`
	NameSpace string `yaml:"namespace"`
}

// Provider contains an identity providers type specific provider data
type Provider struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	File       string `json:"file"`
	CA         string `json:"ca"`
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
}

// IdentityProvider stroes an identity provider
type IdentityProvider struct {
	Kind            string
	APIVersion      string
	MappingMethod   string
	Name            string
	Provider        runtime.RawExtension
	HTFileName      string
	HTFileData      []byte
	CAData          []byte
	CrtData         []byte
	KeyData         []byte
	UseAsChallenger bool
	UseAsLogin      bool
}

const (
	// APIVersion is the apiVersion string
	APIVersion = "config.openshift.io/v1"
	// OAuthNamespace is namespace for oauth manifests
	OAuthNamespace = "openshift-config"
)

// Translate converts OCPv3 OAuth to OCPv4 OAuth Custom Resources
func Translate(identityProviders []IdentityProvider) (*CRD, []*secrets.Secret, []*configmaps.ConfigMap, error) {
	var err error
	var idP interface{}
	var secretsSlice []*secrets.Secret
	var сonfigMapSlice []*configmaps.ConfigMap

	var oauthCrd CRD
	oauthCrd.APIVersion = APIVersion
	oauthCrd.Kind = "OAuth"
	oauthCrd.Metadata.Name = "cluster"
	oauthCrd.Metadata.NameSpace = OAuthNamespace
	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	for _, p := range identityProviders {
		var secret, certSecret, keySecret *secrets.Secret
		var caConfigMap *configmaps.ConfigMap

		p.Provider.Object, _, err = serializer.Decode(p.Provider.Raw, nil, nil)
		if err != nil {
			return nil, nil, nil, err
		}

		kind := p.Kind

		switch kind {
		case "GitHubIdentityProvider":
			idP, secret, caConfigMap, err = buildGitHubIP(serializer, p)
		case "GitLabIdentityProvider":
			idP, secret, caConfigMap, err = buildGitLabIP(serializer, p)
		case "GoogleIdentityProvider":
			idP, secret, err = buildGoogleIP(serializer, p)
		case "HTPasswdPasswordIdentityProvider":
			idP, secret, err = buildHTPasswdIP(serializer, p)
		case "OpenIDIdentityProvider":
			idP, secret, err = buildOpenIDIP(serializer, p)
		case "RequestHeaderIdentityProvider":
			idP, caConfigMap, err = buildRequestHeaderIP(serializer, p)
		case "LDAPPasswordIdentityProvider":
			idP, caConfigMap, err = buildLdapIP(serializer, p)
		case "KeystonePasswordIdentityProvider":
			idP, certSecret, keySecret, caConfigMap, err = buildKeystoneIP(serializer, p)
		case "BasicAuthPasswordIdentityProvider":
			idP, certSecret, keySecret, caConfigMap, err = buildBasicAuthIP(serializer, p)
		default:
			logrus.Infof("Can't handle %s OAuth kind", kind)
			continue
		}

		// Skip OAuth provider if error was returned
		if err != nil {
			logrus.Error("Can't handle ", kind, " skipping.. error:", err)
			continue
		}

		// Check if secret is not empty
		if secret != nil {
			secretsSlice = append(secretsSlice, secret)
		}

		// Check if certSecret is not empty
		if certSecret != nil {
			secretsSlice = append(secretsSlice, certSecret)
			secretsSlice = append(secretsSlice, keySecret)
		}

		// Check if config map is not empty
		if caConfigMap != nil {
			сonfigMapSlice = append(сonfigMapSlice, caConfigMap)
		}

		oauthCrd.Spec.IdentityProviders = append(oauthCrd.Spec.IdentityProviders, idP)
	}

	return &oauthCrd, secretsSlice, сonfigMapSlice, nil
}

// GenYAML returns a YAML of the CRD
func (oauth *CRD) GenYAML() ([]byte, error) {
	yamlBytes, err := yaml.Marshal(&oauth)
	if err != nil {
		logrus.Debugf("Error in OAuth CRD, OAuth CRD - %+v", yamlBytes)
		return nil, err
	}

	return yamlBytes, nil
}
