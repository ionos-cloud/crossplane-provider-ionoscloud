package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/golang/mock/gomock"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	usermock "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/user"
)

func TestUserObserve(t *testing.T) {
	var (
		ctx    = context.Background()
		ctrl   = gomock.NewController(t)
		client = usermock.NewMockClient(ctrl)
		g      = NewWithT(t)
		eu     = externalUser{
			service: client,
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
			cr:       &v1alpha1.User{},
			expectedObservation: managed.ExternalObservation{
				ResourceExists: false,
			},
		},
		{
			scenario: "Resource not found on ionoscloud api",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusNotFound})
				client.EXPECT().GetUser(ctx, gomock.Any()).Return(ionoscloud.User{}, apires, nil)
			},
			cr: &v1alpha1.User{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: idInTest,
				}},
			},
			expectedObservation: managed.ExternalObservation{
				ResourceExists: false,
			},
		},
		{
			scenario: "Error from ionoscloud api sdk when fetching the user",
			mock: func() {
				err := errors.New("internal error")
				client.EXPECT().GetUser(ctx, gomock.Any()).Return(ionoscloud.User{}, nil, err)
			},
			cr: &v1alpha1.User{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: idInTest,
				}},
			},
			errContains: "failed to get user by id",
		},
		{
			scenario: "User exists on ionoscloud",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusOK})
				user := ionoscloud.User{
					Id: pointer.String(idInTest),
					Properties: &ionoscloud.UserProperties{
						Email:             pointer.String("xplane-user@ionoscloud.io"),
						S3CanonicalUserId: pointer.String("400c7ccfed0d"),
					},
				}
				client.EXPECT().GetUser(ctx, gomock.Any()).Return(user, apires, nil)
			},
			cr: &v1alpha1.User{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: idInTest,
				}},
				Spec: v1alpha1.UserSpec{
					ForProvider: userParams(defaultParams),
				},
			},
			expectations: func(mg resource.Managed) {
				cr := mg.(*v1alpha1.User)
				g.Expect(cr.Status.AtProvider.UserID).To(Equal(idInTest))
				g.Expect(cr.Status.AtProvider.S3CanonicalUserID).To(Equal("400c7ccfed0d"))
				g.Expect(xpv1.Available().Equal(cr.GetCondition(xpv1.TypeReady))).To(BeTrue())
			},
			expectedObservation: managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: false,
				ConnectionDetails: managed.ConnectionDetails{
					"email":    []byte("xplane-user@ionoscloud.io"),
					"password": []byte("pwned"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			res, err := eu.Observe(ctx, test.cr)
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

const idInTest = "a8ba2f37-6207-47f8-ab52-fe82021d3259"

var defaultParams func(*v1alpha1.UserParameters) = nil

func userParams(mod func(*v1alpha1.UserParameters)) v1alpha1.UserParameters {
	p := &v1alpha1.UserParameters{
		Email:         "xplane-user@ionoscloud.io",
		FirstName:     "user name",
		LastName:      "test",
		Administrator: false,
		ForceSecAuth:  false,
		Password:      "pwned",
		SecAuthActive: false,
		Active:        false,
		GroupIDs:      []string{"0194bffb-070e-464c-8c5a-4d476489e5e7"},
	}
	if mod != nil {
		mod(p)
	}
	return *p
}
