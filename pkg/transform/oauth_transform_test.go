package transform

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/fusor/cpma/pkg/transform/configmaps"

	"github.com/fusor/cpma/pkg/transform/oauth"
	"github.com/fusor/cpma/pkg/transform/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configv1 "github.com/openshift/api/legacyconfig/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func loadTestIdentityProviders() []oauth.IdentityProvider {
	// TODO: Something is broken here in a way that it's causing the translaters
	// to fail. Need some help with creating test identiy providers in a way
	// that won't crash the translator

	// Build example identity providers, this is straight copy pasted from
	// oauth test, IMO this loading of example identity providers should be
	// some shared test helper
	file := "testdata/bulk-test-master-config.yaml" // File copied into transform pkg testdata
	content, _ := ioutil.ReadFile(file)
	serializer := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var masterV3 configv1.MasterConfig
	_, _, _ = serializer.Decode(content, nil, &masterV3)

	var identityProviders []oauth.IdentityProvider
	for _, identityProvider := range masterV3.OAuthConfig.IdentityProviders {
		providerJSON, _ := identityProvider.Provider.MarshalJSON()
		provider := oauth.Provider{}
		json.Unmarshal(providerJSON, &provider)

		identityProviders = append(identityProviders,
			oauth.IdentityProvider{
				Kind:            provider.Kind,
				APIVersion:      provider.APIVersion,
				MappingMethod:   identityProvider.MappingMethod,
				Name:            identityProvider.Name,
				Provider:        identityProvider.Provider,
				HTFileName:      provider.File,
				HTFileData:      nil,
				CrtData:         nil,
				KeyData:         nil,
				UseAsChallenger: identityProvider.UseAsChallenger,
				UseAsLogin:      identityProvider.UseAsLogin,
			})
	}
	return identityProviders
}

func TestOAuthExtractionTransform(t *testing.T) {
	var expectedManifests []Manifest

	var expectedCrd oauth.CRD
	expectedCrd.APIVersion = "config.openshift.io/v1"
	expectedCrd.Kind = "OAuth"
	expectedCrd.Metadata.Name = "cluster"
	expectedCrd.Metadata.NameSpace = oauth.OAuthNamespace

	var basicAuthIDP oauth.IdentityProviderBasicAuth
	basicAuthIDP.Type = "BasicAuth"
	basicAuthIDP.Challenge = true
	basicAuthIDP.Login = true
	basicAuthIDP.Name = "my_remote_basic_auth_provider"
	basicAuthIDP.MappingMethod = "claim"
	basicAuthIDP.BasicAuth.URL = "https://www.example.com/"
	basicAuthIDP.BasicAuth.TLSClientCert = &oauth.TLSClientCert{Name: "my_remote_basic_auth_provider-client-cert-secret"}
	basicAuthIDP.BasicAuth.TLSClientKey = &oauth.TLSClientKey{Name: "my_remote_basic_auth_provider-client-key-secret"}
	basicAuthIDP.BasicAuth.CA = &oauth.CA{Name: "basicauth-configmap"}

	var basicAuthCrtSecretCrd secrets.Secret
	basicAuthCrtSecretCrd.APIVersion = "v1"
	basicAuthCrtSecretCrd.Kind = "Secret"
	basicAuthCrtSecretCrd.Type = "Opaque"
	basicAuthCrtSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	basicAuthCrtSecretCrd.Metadata.Name = "my_remote_basic_auth_provider-client-cert-secret"
	basicAuthCrtSecretCrd.Data = secrets.BasicAuthFileSecret{BasicAuth: base64.StdEncoding.EncodeToString([]byte(""))}

	var basicAuthKeySecretCrd secrets.Secret
	basicAuthKeySecretCrd.APIVersion = "v1"
	basicAuthKeySecretCrd.Kind = "Secret"
	basicAuthKeySecretCrd.Type = "Opaque"
	basicAuthKeySecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	basicAuthKeySecretCrd.Metadata.Name = "my_remote_basic_auth_provider-client-key-secret"
	basicAuthKeySecretCrd.Data = secrets.BasicAuthFileSecret{BasicAuth: base64.StdEncoding.EncodeToString([]byte(""))}

	var basicAuthConfigMap configmaps.ConfigMap
	basicAuthConfigMap.APIVersion = "v1"
	basicAuthConfigMap.Kind = "ConfigMap"
	basicAuthConfigMap.Metadata.Name = "basicauth-configmap"
	basicAuthConfigMap.Metadata.Namespace = oauth.OAuthNamespace
	basicAuthConfigMap.Data.CAData = ""

	var githubIDP oauth.IdentityProviderGitHub
	githubIDP.Type = "GitHub"
	githubIDP.Challenge = false
	githubIDP.Login = true
	githubIDP.MappingMethod = "claim"
	githubIDP.Name = "github123456789"
	githubIDP.GitHub.HostName = "test.example.com"
	githubIDP.GitHub.CA = &oauth.CA{Name: "github-configmap"}
	githubIDP.GitHub.ClientID = "2d85ea3f45d6777bffd7"
	githubIDP.GitHub.Organizations = []string{"myorganization1", "myorganization2"}
	githubIDP.GitHub.Teams = []string{"myorganization1/team-a", "myorganization2/team-b"}
	githubIDP.GitHub.ClientSecret.Name = "github123456789-secret"

	var githubSecretCrd secrets.Secret
	githubSecretCrd.APIVersion = "v1"
	githubSecretCrd.Kind = "Secret"
	githubSecretCrd.Type = "Opaque"
	githubSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	githubSecretCrd.Metadata.Name = "github123456789-secret"
	githubSecretCrd.Data = secrets.LiteralSecret{ClientSecret: base64.StdEncoding.EncodeToString([]byte("e16a59ad33d7c29fd4354f46059f0950c609a7ea"))}

	var githubConfigMap configmaps.ConfigMap
	githubConfigMap.APIVersion = "v1"
	githubConfigMap.Kind = "ConfigMap"
	githubConfigMap.Metadata.Name = "github-configmap"
	githubConfigMap.Metadata.Namespace = oauth.OAuthNamespace
	githubConfigMap.Data.CAData = ""

	var gitlabIDP oauth.IdentityProviderGitLab
	gitlabIDP.Name = "gitlab123456789"
	gitlabIDP.Type = "GitLab"
	gitlabIDP.Challenge = true
	gitlabIDP.Login = true
	gitlabIDP.MappingMethod = "claim"
	gitlabIDP.GitLab.URL = "https://gitlab.com/"
	gitlabIDP.GitLab.CA = &oauth.CA{Name: "gitlab-configmap"}
	gitlabIDP.GitLab.ClientID = "fake-id"
	gitlabIDP.GitLab.ClientSecret.Name = "gitlab123456789-secret"

	var gitlabSecretCrd secrets.Secret
	gitlabSecretCrd.APIVersion = "v1"
	gitlabSecretCrd.Kind = "Secret"
	gitlabSecretCrd.Type = "Opaque"
	gitlabSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	gitlabSecretCrd.Metadata.Name = "gitlab123456789-secret"
	gitlabSecretCrd.Data = secrets.LiteralSecret{ClientSecret: "fake-secret"}

	var gitlabConfigMap configmaps.ConfigMap
	gitlabConfigMap.APIVersion = "v1"
	gitlabConfigMap.Kind = "ConfigMap"
	gitlabConfigMap.Metadata.Name = "gitlab-configmap"
	gitlabConfigMap.Metadata.Namespace = oauth.OAuthNamespace
	gitlabConfigMap.Data.CAData = ""

	var googleIDP oauth.IdentityProviderGoogle
	googleIDP.Type = "Google"
	googleIDP.Challenge = false
	googleIDP.Login = true
	googleIDP.MappingMethod = "claim"
	googleIDP.Name = "google123456789123456789"
	googleIDP.Google.ClientID = "82342890327-tf5lqn4eikdf4cb4edfm85jiqotvurpq.apps.googleusercontent.com"
	googleIDP.Google.ClientSecret.Name = "google123456789123456789-secret"
	googleIDP.Google.HostedDomain = "test.example.com"

	var googleSecretCrd secrets.Secret
	googleSecretCrd.APIVersion = "v1"
	googleSecretCrd.Kind = "Secret"
	googleSecretCrd.Type = "Opaque"
	googleSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	googleSecretCrd.Metadata.Name = "google123456789123456789-secret"
	googleSecretCrd.Data = secrets.LiteralSecret{ClientSecret: "e16a59ad33d7c29fd4354f46059f0950c609a7ea"}

	var keystoneIDP oauth.IdentityProviderKeystone
	keystoneIDP.Type = "Keystone"
	keystoneIDP.Challenge = true
	keystoneIDP.Login = true
	keystoneIDP.Name = "my_keystone_provider"
	keystoneIDP.MappingMethod = "claim"
	keystoneIDP.Keystone.DomainName = "default"
	keystoneIDP.Keystone.URL = "http://fake.url:5000"
	keystoneIDP.Keystone.CA = &oauth.CA{Name: "keystone-configmap"}
	keystoneIDP.Keystone.TLSClientCert = &oauth.TLSClientCert{Name: "my_keystone_provider-client-cert-secret"}
	keystoneIDP.Keystone.TLSClientKey = &oauth.TLSClientKey{Name: "my_keystone_provider-client-key-secret"}

	var keystoneConfigMap configmaps.ConfigMap
	keystoneConfigMap.APIVersion = "v1"
	keystoneConfigMap.Kind = "ConfigMap"
	keystoneConfigMap.Metadata.Name = "keystone-configmap"
	keystoneConfigMap.Metadata.Namespace = oauth.OAuthNamespace
	keystoneConfigMap.Data.CAData = ""

	var htpasswdIDP oauth.IdentityProviderHTPasswd
	htpasswdIDP.Name = "htpasswd_auth"
	htpasswdIDP.Type = "HTPasswd"
	htpasswdIDP.Challenge = true
	htpasswdIDP.Login = true
	htpasswdIDP.MappingMethod = "claim"
	htpasswdIDP.HTPasswd.FileData.Name = "htpasswd_auth-secret"

	var htpasswdSecretCrd secrets.Secret
	htpasswdSecretCrd.APIVersion = "v1"
	htpasswdSecretCrd.Kind = "Secret"
	htpasswdSecretCrd.Type = "Opaque"
	htpasswdSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	htpasswdSecretCrd.Metadata.Name = "htpasswd_auth-secret"
	htpasswdSecretCrd.Data = secrets.HTPasswdFileSecret{HTPasswd: ""}

	var keystoneCrtSecretCrd secrets.Secret
	keystoneCrtSecretCrd.APIVersion = "v1"
	keystoneCrtSecretCrd.Kind = "Secret"
	keystoneCrtSecretCrd.Type = "Opaque"
	keystoneCrtSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	keystoneCrtSecretCrd.Metadata.Name = "my_keystone_provider-client-cert-secret"
	keystoneCrtSecretCrd.Data = secrets.KeystoneFileSecret{Keystone: ""}

	var keystoneKeySecretCrd secrets.Secret
	keystoneKeySecretCrd.APIVersion = "v1"
	keystoneKeySecretCrd.Kind = "Secret"
	keystoneKeySecretCrd.Type = "Opaque"
	keystoneKeySecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	keystoneKeySecretCrd.Metadata.Name = "my_keystone_provider-client-key-secret"
	keystoneKeySecretCrd.Data = secrets.KeystoneFileSecret{Keystone: ""}

	var ldapIDP oauth.IdentityProviderLDAP
	ldapIDP.Name = "my_ldap_provider"
	ldapIDP.Type = "LDAP"
	ldapIDP.Challenge = true
	ldapIDP.Login = true
	ldapIDP.MappingMethod = "claim"
	ldapIDP.LDAP.Attributes.ID = []string{"dn"}
	ldapIDP.LDAP.Attributes.Email = []string{"mail"}
	ldapIDP.LDAP.Attributes.Name = []string{"cn"}
	ldapIDP.LDAP.Attributes.PreferredUsername = []string{"uid"}
	ldapIDP.LDAP.BindDN = "123"
	ldapIDP.LDAP.BindPassword = "321"
	ldapIDP.LDAP.CA = &oauth.CA{Name: "ldap-configmap"}
	ldapIDP.LDAP.Insecure = false
	ldapIDP.LDAP.URL = "ldap://ldap.example.com/ou=users,dc=acme,dc=com?uid"

	var ldapConfigMap configmaps.ConfigMap
	ldapConfigMap.APIVersion = "v1"
	ldapConfigMap.Kind = "ConfigMap"
	ldapConfigMap.Metadata.Name = "ldap-configmap"
	ldapConfigMap.Metadata.Namespace = oauth.OAuthNamespace
	ldapConfigMap.Data.CAData = ""

	var requestHeaderIDP oauth.IdentityProviderRequestHeader
	requestHeaderIDP.Type = "RequestHeader"
	requestHeaderIDP.Name = "my_request_header_provider"
	requestHeaderIDP.Challenge = true
	requestHeaderIDP.Login = true
	requestHeaderIDP.MappingMethod = "claim"
	requestHeaderIDP.RequestHeader.ChallengeURL = "https://example.com"
	requestHeaderIDP.RequestHeader.LoginURL = "https://example.com"
	requestHeaderIDP.RequestHeader.CA = &oauth.CA{Name: "requestheader-configmap"}
	requestHeaderIDP.RequestHeader.ClientCommonNames = []string{"my-auth-proxy"}
	requestHeaderIDP.RequestHeader.Headers = []string{"X-Remote-User", "SSO-User"}
	requestHeaderIDP.RequestHeader.EmailHeaders = []string{"X-Remote-User-Email"}
	requestHeaderIDP.RequestHeader.NameHeaders = []string{"X-Remote-User-Display-Name"}
	requestHeaderIDP.RequestHeader.PreferredUsernameHeaders = []string{"X-Remote-User-Login"}

	var requestheaderConfigMap configmaps.ConfigMap
	requestheaderConfigMap.APIVersion = "v1"
	requestheaderConfigMap.Kind = "ConfigMap"
	requestheaderConfigMap.Metadata.Name = "requestheader-configmap"
	requestheaderConfigMap.Metadata.Namespace = oauth.OAuthNamespace
	requestheaderConfigMap.Data.CAData = ""

	var openidIDP oauth.IdentityProviderOpenID
	openidIDP.Type = "OpenID"
	openidIDP.Challenge = false
	openidIDP.Login = true
	openidIDP.MappingMethod = "claim"
	openidIDP.Name = "my_openid_connect"
	openidIDP.OpenID.ClientID = "testid"
	openidIDP.OpenID.Claims.PreferredUsername = []string{"preferred_username", "email"}
	openidIDP.OpenID.Claims.Name = []string{"nickname", "given_name", "name"}
	openidIDP.OpenID.Claims.Email = []string{"custom_email_claim", "email"}
	openidIDP.OpenID.URLs.Authorize = "https://myidp.example.com/oauth2/authorize"
	openidIDP.OpenID.URLs.Token = "https://myidp.example.com/oauth2/token"
	openidIDP.OpenID.ClientSecret.Name = "my_openid_connect-secret"

	var openidSecretCrd secrets.Secret
	openidSecretCrd.APIVersion = "v1"
	openidSecretCrd.Kind = "Secret"
	openidSecretCrd.Type = "Opaque"
	openidSecretCrd.Metadata.Namespace = oauth.OAuthNamespace
	openidSecretCrd.Metadata.Name = "my_openid_connect-secret"
	openidSecretCrd.Data = secrets.LiteralSecret{ClientSecret: "testsecret"}

	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, basicAuthIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, githubIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, gitlabIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, googleIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, htpasswdIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, keystoneIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, ldapIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, requestHeaderIDP)
	expectedCrd.Spec.IdentityProviders = append(expectedCrd.Spec.IdentityProviders, openidIDP)

	expectedManifest, err := expectedCrd.GenYAML()
	require.NoError(t, err)
	basicAuthCrtSecretManifest, err := basicAuthCrtSecretCrd.GenYAML()
	require.NoError(t, err)
	basicAuthKeySecretManifest, err := basicAuthKeySecretCrd.GenYAML()
	require.NoError(t, err)
	githubSecretManifest, err := githubSecretCrd.GenYAML()
	require.NoError(t, err)
	gitlabSecretManifest, err := gitlabSecretCrd.GenYAML()
	require.NoError(t, err)
	googleSecretManifest, err := googleSecretCrd.GenYAML()
	require.NoError(t, err)
	htpasswdSecretManifest, err := htpasswdSecretCrd.GenYAML()
	require.NoError(t, err)
	keystoneCrtSecretManifest, err := keystoneCrtSecretCrd.GenYAML()
	require.NoError(t, err)
	keystoneKeySecretManifest, err := keystoneKeySecretCrd.GenYAML()
	require.NoError(t, err)
	openidSecretManifest, err := openidSecretCrd.GenYAML()
	require.NoError(t, err)

	basicAuthConfigMapManifest, err := basicAuthConfigMap.GenYAML()
	require.NoError(t, err)
	githubConfigMapManifest, err := githubConfigMap.GenYAML()
	require.NoError(t, err)
	gitlabConfigMapManifest, err := gitlabConfigMap.GenYAML()
	require.NoError(t, err)
	keystoneConfigMapManifest, err := keystoneConfigMap.GenYAML()
	require.NoError(t, err)
	ldapConfigMapManifest, err := ldapConfigMap.GenYAML()
	require.NoError(t, err)
	requestheaderConfigMapManifest, err := requestheaderConfigMap.GenYAML()
	require.NoError(t, err)

	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-oauth.yaml", CRD: expectedManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-my_remote_basic_auth_provider-client-cert-secret.yaml", CRD: basicAuthCrtSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-my_remote_basic_auth_provider-client-key-secret.yaml", CRD: basicAuthKeySecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-github123456789-secret.yaml", CRD: githubSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-gitlab123456789-secret.yaml", CRD: gitlabSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-google123456789123456789-secret.yaml", CRD: googleSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-htpasswd_auth-secret.yaml", CRD: htpasswdSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-my_keystone_provider-client-cert-secret.yaml", CRD: keystoneCrtSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-my_keystone_provider-client-key-secret.yaml", CRD: keystoneKeySecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-secret-my_openid_connect-secret.yaml", CRD: openidSecretManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-configmap-basicauth-configmap.yaml", CRD: basicAuthConfigMapManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-configmap-github-configmap.yaml", CRD: githubConfigMapManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-configmap-gitlab-configmap.yaml", CRD: gitlabConfigMapManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-configmap-keystone-configmap.yaml", CRD: keystoneConfigMapManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-configmap-ldap-configmap.yaml", CRD: ldapConfigMapManifest})
	expectedManifests = append(expectedManifests,
		Manifest{Name: "100_CPMA-cluster-config-configmap-requestheader-configmap.yaml", CRD: requestheaderConfigMapManifest})

	testCases := []struct {
		name              string
		expectedManifests []Manifest
	}{
		{
			name:              "transform registries extraction",
			expectedManifests: expectedManifests,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualManifestsChan := make(chan []Manifest)

			// Override flush method
			manifestOutputFlush = func(manifests []Manifest) error {
				actualManifestsChan <- manifests
				return nil
			}

			testExtraction := OAuthExtraction{
				IdentityProviders: loadTestIdentityProviders(),
			}

			go func() {
				transformOutput, err := testExtraction.Transform()
				if err != nil {
					t.Error(err)
				}
				transformOutput.Flush()
			}()

			actualManifests := <-actualManifestsChan
			assert.Equal(t, actualManifests, tc.expectedManifests)
		})
	}
}
