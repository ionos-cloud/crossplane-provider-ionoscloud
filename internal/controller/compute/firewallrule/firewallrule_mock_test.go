//go:build firewallrule_mock

package firewallrule

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	fwClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/firewallrule"
	ipblockClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake FirewallRule service
// ---------------------------------------------------------------------------

type storedFirewallRule struct {
	rule         sdkgo.FirewallRule
	datacenterID string
	serverID     string
	nicID        string
}

type fakeFirewallRuleService struct {
	mocktest.FakeServiceBase
	rules  map[string]storedFirewallRule
	nextID int
}

func newFakeFirewallRuleService(serverURL string) *fakeFirewallRuleService {
	return &fakeFirewallRuleService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		rules:           make(map[string]storedFirewallRule),
		nextID:          1,
	}
}

func (f *fakeFirewallRuleService) key(dcID, srvID, nicID, fwID string) string {
	return dcID + "/" + srvID + "/" + nicID + "/" + fwID
}

func (f *fakeFirewallRuleService) CheckDuplicateFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwName, protocol string) (*sdkgo.FirewallRule, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sr := range f.rules {
		if sr.datacenterID == datacenterID && sr.serverID == serverID && sr.nicID == nicID &&
			sr.rule.Properties != nil && sr.rule.Properties.Name != nil && *sr.rule.Properties.Name == fwName {
			r := sr.rule
			return &r, nil
		}
	}
	return nil, nil
}

func (f *fakeFirewallRuleService) GetFirewallRuleID(r *sdkgo.FirewallRule) (string, error) {
	if r != nil {
		if id, ok := r.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting firewallrule id")
	}
	return "", nil
}

func (f *fakeFirewallRuleService) GetFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwID string) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.FirewallRule{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	sr, ok := f.rules[f.key(datacenterID, serverID, nicID, fwID)]
	if !ok {
		return sdkgo.FirewallRule{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("firewallrule %s not found", fwID)
	}
	return sr.rule, mocktest.OKResponse(), nil
}

func (f *fakeFirewallRuleService) CreateFirewallRule(_ context.Context, datacenterID, serverID, nicID string, r sdkgo.FirewallRule) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.FirewallRule{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}
	fwID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.FirewallruleProperties{}
	if r.Properties != nil {
		if r.Properties.Name != nil {
			props.Name = r.Properties.Name
		}
		if r.Properties.Protocol != nil {
			props.Protocol = r.Properties.Protocol
		}
	}
	newRule := sdkgo.FirewallRule{
		Id:         &fwID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.rules[f.key(datacenterID, serverID, nicID, fwID)] = storedFirewallRule{rule: newRule, datacenterID: datacenterID, serverID: serverID, nicID: nicID}

	return newRule, f.AcceptedResponse("create", fwID), nil
}

func (f *fakeFirewallRuleService) UpdateFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwID string, props sdkgo.FirewallruleProperties) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.FirewallRule{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}
	sr, ok := f.rules[f.key(datacenterID, serverID, nicID, fwID)]
	if !ok {
		return sdkgo.FirewallRule{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("firewallrule %s not found", fwID)
	}
	if props.Name != nil {
		sr.rule.Properties.Name = props.Name
	}
	if props.Protocol != nil {
		sr.rule.Properties.Protocol = props.Protocol
	}
	f.rules[f.key(datacenterID, serverID, nicID, fwID)] = sr

	return sr.rule, f.AcceptedResponse("update", fwID), nil
}

func (f *fakeFirewallRuleService) DeleteFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}
	f.DeleteCalls = append(f.DeleteCalls, fwID)
	delete(f.rules, f.key(datacenterID, serverID, nicID, fwID))

	return f.AcceptedResponse("delete", fwID), nil
}

// Test-only helpers

func (f *fakeFirewallRuleService) storeRule(dcID, srvID, nicID, fwID string, r sdkgo.FirewallRule) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.rules[f.key(dcID, srvID, nicID, fwID)] = storedFirewallRule{rule: r, datacenterID: dcID, serverID: srvID, nicID: nicID}
}

func (f *fakeFirewallRuleService) removeRule(dcID, srvID, nicID, fwID string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.rules, f.key(dcID, srvID, nicID, fwID))
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorFirewallRule struct {
	service        fwClient.Client
	ipBlockService ipblockClient.Client
	log            logging.Logger
}

func (c *testConnectorFirewallRule) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalFirewallRule{service: c.service, ipBlockService: c.ipBlockService, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeFirewallRuleService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestFirewallRuleController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FirewallRule Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeFirewallRuleService(testServer.URL())
	fakeIPSvc := mocktest.NewStubIPBlockService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.FirewallRuleGroupKind,
		GroupVersionKind: v1alpha1.FirewallRuleGroupVersionKind,
		ManagedResource:  &v1alpha1.FirewallRule{},
		ManagedList:      &v1alpha1.FirewallRuleList{},
		Connector: &testConnectorFirewallRule{
			service:        fakeSvc,
			ipBlockService: fakeIPSvc,
			log:            logging.NewLogrLogger(mocktest.Logger),
		},
	})
	k8sClient = harness.K8sClient
})

var _ = AfterSuite(func() {
	mocktest.TeardownEnvTest(harness, testServer)
})

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newFirewallRuleCR(name string) *v1alpha1.FirewallRule {
	return &v1alpha1.FirewallRule{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.FirewallRuleSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.FirewallRuleParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: mocktest.TestDatacenterID},
				ServerCfg:     v1alpha1.ServerConfig{ServerID: mocktest.TestServerID},
				NicCfg:        v1alpha1.NicConfig{NicID: mocktest.TestNicID},
				Protocol:      "TCP",
				Name:          name,
			},
		},
	}
}

func getFirewallRuleCR(ctx context.Context, name string) (*v1alpha1.FirewallRule, error) {
	cr := &v1alpha1.FirewallRule{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("FirewallRule Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "FirewallRule",
		CRPrefix:     "test-fw",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newFirewallRuleCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newFirewallRuleCR(name)
			cr.Annotations = map[string]string{
				"ionos.cloud/post-request-id": annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getFirewallRuleCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.FirewallRule).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.FirewallRule).Status.AtProvider.FirewallRuleID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.FirewallRule)
			cr.Spec.ForProvider.Name = "updated-firewallrule"
		},
		StoreResource: func(externalID string) {
			fakeSvc.storeRule(mocktest.TestDatacenterID, mocktest.TestServerID, mocktest.TestNicID, externalID, sdkgo.FirewallRule{
				Id:         ptr.To(externalID),
				Properties: &sdkgo.FirewallruleProperties{Name: ptr.To("isreqdone-fw"), Protocol: ptr.To("TCP")},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeRule(mocktest.TestDatacenterID, mocktest.TestServerID, mocktest.TestNicID, externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})
})
