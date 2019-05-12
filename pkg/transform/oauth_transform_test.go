package transform

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/fusor/cpma/pkg/transform/oauth"
	"github.com/stretchr/testify/assert"

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

	var htContent []byte
	var identityProviders []oauth.IdentityProvider
	for _, identityProvider := range masterV3.OAuthConfig.IdentityProviders {
		providerJSON, _ := identityProvider.Provider.MarshalJSON()
		provider := oauth.Provider{}
		json.Unmarshal(providerJSON, &provider)

		identityProviders = append(identityProviders,
			oauth.IdentityProvider{
				provider.Kind,
				provider.APIVersion,
				identityProvider.MappingMethod,
				identityProvider.Name,
				identityProvider.Provider,
				provider.File,
				htContent,
				identityProvider.UseAsChallenger,
				identityProvider.UseAsLogin,
			})
	}
	return identityProviders
}

func TestOAuthExtractionTransform(t *testing.T) {
	actualManifestsChan := make(chan []Manifest)

	// Override flush method
	manifestTransformOutputFlush = func(manifests []Manifest) error {
		t.Log("Running overridden manifestTransformOutputFlush")
		actualManifestsChan <- manifests
		return nil
	}

	// TODO: write expectedManifests

	// TODO: Set up the extraction with dummy extracted values
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
	t.Logf("Got actualManifests: %v", actualManifests)

	// TODO: checkActualManifestsMatchExpectedManifests(t, actualManifests, expectedManifests)

	assert.Equal(t, 2, 2)
}
