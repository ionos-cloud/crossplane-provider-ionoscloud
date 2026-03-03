package datacenter

import (
	"context"
	"errors"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	datacentermock "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/datacenter"
)

const datacenterIDInTest = "b3a07cf8-9c18-4fda-8a2f-1234567890ab"

func TestDatacenterObserve(t *testing.T) {
	var (
		ctx  = context.Background()
		ctrl = gomock.NewController(t)
		mc   = datacentermock.NewMockClient(ctrl)
		g    = NewWithT(t)
		ed   = externalDatacenter{
			service: mc,
			log:     logging.NewNopLogger(),
		}
	)

	tests := []struct {
		scenario            string
		cr                  resource.Managed
		expectations        func(resource.Managed)
		expectedObservation managed.ExternalObservation
		mock                func()
		errContains         string
	}{
		{
			scenario: "A resource without external name doesn't exist",
			cr:       &v1alpha1.Datacenter{},
			expectedObservation: managed.ExternalObservation{
				ResourceExists: false,
			},
		},
		{
			scenario: "Resource not found on ionoscloud api",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusNotFound})
				mc.EXPECT().GetDatacenter(ctx, gomock.Any()).Return(ionoscloud.Datacenter{}, apires, errors.New("404 not found"))
			},
			cr: &v1alpha1.Datacenter{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: datacenterIDInTest,
				},
			}},
			expectedObservation: managed.ExternalObservation{
				ResourceExists: false,
			},
		},
		{
			scenario: "API ionoscloud returns an error when fetching the datacenter",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusInternalServerError})
				mc.EXPECT().GetDatacenter(ctx, gomock.Any()).Return(ionoscloud.Datacenter{}, apires, errors.New("internal error"))
			},
			cr: &v1alpha1.Datacenter{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: datacenterIDInTest,
				},
			}},
			errContains: "failed to get datacenter by id",
		},
		{
			scenario: "Datacenter found and properties are up to date",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusOK})
				dc := ionoscloud.Datacenter{
					Id: ptr.To(datacenterIDInTest),
					Properties: &ionoscloud.DatacenterProperties{
						Name:        ptr.To("my-datacenter"),
						Description: ptr.To("test description"),
					},
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ptr.To("AVAILABLE"),
					},
				}
				mc.EXPECT().GetDatacenter(ctx, gomock.Any()).Return(dc, apires, nil)
				mc.EXPECT().GetCPUFamiliesForDatacenter(ctx, datacenterIDInTest).Return([]string{"AMD_OPTERON"}, nil)
			},
			cr: &v1alpha1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: datacenterIDInTest,
					},
				},
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:        "my-datacenter",
						Description: "test description",
						Location:    "de/fra",
					},
				},
			},
			expectations: func(mg resource.Managed) {
				cr := mg.(*v1alpha1.Datacenter)
				g.Expect(cr.Status.AtProvider.DatacenterID).To(Equal(datacenterIDInTest))
				g.Expect(cr.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(cr.Status.AtProvider.AvailableCPUFamilies).To(ContainElement("AMD_OPTERON"))
			},
			expectedObservation: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
		{
			scenario: "Datacenter found but properties differ (not up to date)",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusOK})
				dc := ionoscloud.Datacenter{
					Id: ptr.To(datacenterIDInTest),
					Properties: &ionoscloud.DatacenterProperties{
						Name:        ptr.To("old-name"),
						Description: ptr.To("test description"),
					},
					Metadata: &ionoscloud.DatacenterElementMetadata{
						State: ptr.To("AVAILABLE"),
					},
				}
				mc.EXPECT().GetDatacenter(ctx, gomock.Any()).Return(dc, apires, nil)
				mc.EXPECT().GetCPUFamiliesForDatacenter(ctx, datacenterIDInTest).Return(nil, nil)
			},
			cr: &v1alpha1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						meta.AnnotationKeyExternalName: datacenterIDInTest,
					},
				},
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:        "new-name",
						Description: "test description",
						Location:    "de/fra",
					},
				},
			},
			expectedObservation: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			res, err := ed.Observe(ctx, test.cr)
			if test.errContains != "" {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
				return
			}
			if test.expectations != nil {
				test.expectations(test.cr)
			}
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(res).To(Equal(test.expectedObservation))
		})
	}
}

func TestDatacenterCreate(t *testing.T) {
	var (
		ctx  = context.Background()
		ctrl = gomock.NewController(t)
		mc   = datacentermock.NewMockClient(ctrl)
		g    = NewWithT(t)
	)

	tests := []struct {
		scenario             string
		cr                   resource.Managed
		expectations         func(resource.Managed)
		expectedCreation     managed.ExternalCreation
		mock                 func()
		errContains          string
		isUniqueNamesEnabled bool
	}{
		{
			scenario: "State is BUSY, returns early with no error",
			cr: &v1alpha1.Datacenter{
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						State: compute.BUSY,
					},
				},
			},
		},
		{
			scenario:             "API returns error on create (unique names disabled)",
			isUniqueNamesEnabled: false,
			mock: func() {
				mc.EXPECT().CreateDatacenter(ctx, gomock.Any()).Return(ionoscloud.Datacenter{}, nil, errors.New("internal error"))
			},
			cr:          &v1alpha1.Datacenter{},
			errContains: "failed to create datacenter",
		},
		{
			scenario:             "Unique names enabled, duplicate found — existing datacenter is imported",
			isUniqueNamesEnabled: true,
			mock: func() {
				existing := &ionoscloud.Datacenter{Id: ptr.To(datacenterIDInTest)}
				mc.EXPECT().CheckDuplicateDatacenter(ctx, "my-datacenter", "de/fra").Return(existing, nil)
				mc.EXPECT().GetDatacenterID(existing).Return(datacenterIDInTest, nil)
			},
			cr: &v1alpha1.Datacenter{
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:     "my-datacenter",
						Location: "de/fra",
					},
				},
			},
			expectations: func(mg resource.Managed) {
				cr := mg.(*v1alpha1.Datacenter)
				g.Expect(cr.Status.AtProvider.DatacenterID).To(Equal(datacenterIDInTest))
				g.Expect(meta.GetExternalName(cr)).To(Equal(datacenterIDInTest))
			},
		},
		{
			scenario:             "Unique names enabled, no duplicate found, create fails",
			isUniqueNamesEnabled: true,
			mock: func() {
				mc.EXPECT().CheckDuplicateDatacenter(ctx, "my-datacenter", "de/fra").Return(nil, nil)
				mc.EXPECT().GetDatacenterID(gomock.Any()).Return("", nil)
				mc.EXPECT().CreateDatacenter(ctx, gomock.Any()).Return(ionoscloud.Datacenter{}, nil, errors.New("quota exceeded"))
			},
			cr: &v1alpha1.Datacenter{
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:     "my-datacenter",
						Location: "de/fra",
					},
				},
			},
			errContains: "failed to create datacenter",
		},
		{
			scenario:             "Unique names enabled, CheckDuplicateDatacenter returns error",
			isUniqueNamesEnabled: true,
			mock: func() {
				mc.EXPECT().CheckDuplicateDatacenter(ctx, "my-datacenter", "de/fra").Return(nil, errors.New("duplicate check failed"))
			},
			cr: &v1alpha1.Datacenter{
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:     "my-datacenter",
						Location: "de/fra",
					},
				},
			},
			errContains: "duplicate check failed",
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			ed := externalDatacenter{
				service:              mc,
				log:                  logging.NewNopLogger(),
				isUniqueNamesEnabled: test.isUniqueNamesEnabled,
			}
			if test.mock != nil {
				test.mock()
			}
			res, err := ed.Create(ctx, test.cr)
			if test.errContains != "" {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
				return
			}
			if test.expectations != nil {
				test.expectations(test.cr)
			}
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(res).To(Equal(test.expectedCreation))
		})
	}
}

func TestDatacenterUpdate(t *testing.T) {
	var (
		ctx  = context.Background()
		ctrl = gomock.NewController(t)
		mc   = datacentermock.NewMockClient(ctrl)
		g    = NewWithT(t)
		ed   = externalDatacenter{
			service: mc,
			log:     logging.NewNopLogger(),
		}
	)

	tests := []struct {
		scenario       string
		cr             resource.Managed
		expectedUpdate managed.ExternalUpdate
		mock           func()
		errContains    string
	}{
		{
			scenario: "State is BUSY, returns early with no error",
			cr: &v1alpha1.Datacenter{
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						State: compute.BUSY,
					},
				},
			},
		},
		{
			scenario: "State is UPDATING, returns early with no error",
			cr: &v1alpha1.Datacenter{
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						State: compute.UPDATING,
					},
				},
			},
		},
		{
			scenario: "API returns error on update",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusInternalServerError})
				mc.EXPECT().UpdateDatacenter(ctx, datacenterIDInTest, gomock.Any()).Return(ionoscloud.Datacenter{}, apires, errors.New("update failed"))
			},
			cr: &v1alpha1.Datacenter{
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:        "my-datacenter",
						Description: "test description",
					},
				},
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						DatacenterID: datacenterIDInTest,
					},
				},
			},
			errContains: "failed to update datacenter",
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			res, err := ed.Update(ctx, test.cr)
			if test.errContains != "" {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
				return
			}
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(res).To(Equal(test.expectedUpdate))
		})
	}
}

func TestDatacenterDelete(t *testing.T) {
	var (
		ctx  = context.Background()
		ctrl = gomock.NewController(t)
		mc   = datacentermock.NewMockClient(ctrl)
		g    = NewWithT(t)
		ed   = externalDatacenter{
			service: mc,
			log:     logging.NewNopLogger(),
		}
	)

	tests := []struct {
		scenario    string
		cr          resource.Managed
		mock        func()
		errContains string
	}{
		{
			scenario: "State is DESTROYING, returns early with no error",
			cr: &v1alpha1.Datacenter{
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						State: compute.DESTROYING,
					},
				},
			},
		},
		{
			scenario: "API returns 500 error on delete",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusInternalServerError})
				mc.EXPECT().DeleteDatacenter(ctx, datacenterIDInTest).Return(apires, errors.New("internal server error"))
			},
			cr: &v1alpha1.Datacenter{
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						DatacenterID: datacenterIDInTest,
					},
				},
			},
			errContains: "failed to delete datacenter",
		},
		{
			scenario: "API returns 404 on delete (already gone — no error)",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusNotFound})
				mc.EXPECT().DeleteDatacenter(ctx, datacenterIDInTest).Return(apires, errors.New("404 not found"))
			},
			cr: &v1alpha1.Datacenter{
				Status: v1alpha1.DatacenterStatus{
					AtProvider: v1alpha1.DatacenterObservation{
						DatacenterID: datacenterIDInTest,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			_, err := ed.Delete(ctx, test.cr)
			if test.errContains != "" {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
				return
			}
			g.Expect(err).ToNot(HaveOccurred())
		})
	}
}
