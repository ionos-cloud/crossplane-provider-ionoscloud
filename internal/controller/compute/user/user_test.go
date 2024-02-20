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
	"k8s.io/utils/ptr"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	usermock "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/mock/clients/compute/user"
)

const userIDInTest = "a8ba2f37-6207-47f8-ab52-fe82021d3259"
const groupIDInTest = "5458a703-6450-4ddd-b133-59349c83f832"

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
					meta.AnnotationKeyExternalName: userIDInTest,
				}},
			},
			expectedObservation: managed.ExternalObservation{
				ResourceExists: false,
			},
		},
		{
			scenario: "API ionoscloud returns an error when fetching the user",
			mock: func() {
				err := errors.New("internal error")
				client.EXPECT().GetUser(ctx, gomock.Any()).Return(ionoscloud.User{}, nil, err)
			},
			cr: &v1alpha1.User{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: userIDInTest,
				}},
			},
			errContains: "failed to get user by id",
		},
		{
			scenario: "User exists on ionoscloud with groups",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusOK})
				user := ionoscloud.User{
					Id: ptr.To(userIDInTest),
					Properties: &ionoscloud.UserProperties{
						Email:             ptr.To("xplane-user@ionoscloud.io"),
						S3CanonicalUserId: ptr.To("400c7ccfed0d"),
						Active:            ptr.To(true),
						SecAuthActive:     ptr.To(true),
						Firstname:         ptr.To("test"),
						Lastname:          ptr.To("user"),
						Administrator:     ptr.To(false),
					},
				}
				client.EXPECT().GetUser(ctx, gomock.Any()).Return(user, apires, nil)
				client.EXPECT().GetUserGroups(ctx, gomock.Any()).Return([]string{"5458a703-6450-4ddd-b133-59349c83f832"}, nil)
			},
			cr: &v1alpha1.User{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					meta.AnnotationKeyExternalName: userIDInTest,
				}},
				Spec: v1alpha1.UserSpec{
					ForProvider: userParams(defaultParams),
				},
			},
			expectations: func(mg resource.Managed) {
				cr := mg.(*v1alpha1.User)
				g.Expect(cr.Status.AtProvider.UserID).To(Equal(userIDInTest))
				g.Expect(cr.Status.AtProvider.S3CanonicalUserID).To(Equal("400c7ccfed0d"))
				g.Expect(cr.Status.AtProvider.GroupIDs).To(ContainElement("5458a703-6450-4ddd-b133-59349c83f832"))
				g.Expect(xpv1.Available().Equal(cr.GetCondition(xpv1.TypeReady))).To(BeTrue())
			},
			expectedObservation: managed.ExternalObservation{
				ResourceExists:          true,
				ResourceUpToDate:        false,
				ResourceLateInitialized: true,
				ConnectionDetails: managed.ConnectionDetails{
					"email":    []byte("xplane-user@ionoscloud.io"),
					"password": []byte("$3cr3t"),
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

func TestUserCreate(t *testing.T) {
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
		expectedObservation managed.ExternalCreation
		mock                func()
		errContains         string
	}{
		{
			scenario: "API ionoscloud returns an error",
			mock: func() {
				err := errors.New("internal error")
				client.EXPECT().CreateUser(ctx, gomock.Any()).Return(ionoscloud.User{}, nil, err)
			},
			cr:          &v1alpha1.User{},
			errContains: "failed to create user",
		},
		{
			scenario: "User creation with groups results in connection details",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusAccepted})
				user := ionoscloud.User{
					Id: ptr.To(userIDInTest),
					Properties: &ionoscloud.UserProperties{
						Email:             ptr.To("xplane-user@ionoscloud.io"),
						Firstname:         ptr.To("user name"),
						Lastname:          ptr.To("test"),
						S3CanonicalUserId: ptr.To("400c7ccfed0d"),
						Administrator:     ptr.To(false),
						ForceSecAuth:      ptr.To(false),
						SecAuthActive:     ptr.To(false),
						Active:            ptr.To(true),
					},
				}
				client.EXPECT().CreateUser(ctx, gomock.Any()).Return(user, apires, nil)
				client.EXPECT().AddUserToGroup(ctx, groupIDInTest, userIDInTest).Return(user, apires, nil)
			},
			cr: &v1alpha1.User{Spec: v1alpha1.UserSpec{
				ForProvider: userParams(defaultParams),
			}},
			expectations: func(mg resource.Managed) {
				cr := mg.(*v1alpha1.User)
				g.Expect(cr.ObjectMeta.Annotations).To(HaveKeyWithValue(meta.AnnotationKeyExternalName, userIDInTest))
				g.Expect(cr.Status.AtProvider.UserID).To(Equal(userIDInTest))
				g.Expect(cr.Status.AtProvider.S3CanonicalUserID).To(Equal("400c7ccfed0d"))
				g.Expect(cr.Status.AtProvider.Active).To(BeTrue())
				g.Expect(cr.Status.AtProvider.SecAuthActive).To(BeFalse())
				g.Expect(cr.GetCondition(xpv1.TypeReady).Equal(xpv1.Creating())).To(BeTrue())
				g.Expect(cr.Spec.ForProvider.Password).To(BeEmpty())
			},
			expectedObservation: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{
					"email":    []byte("xplane-user@ionoscloud.io"),
					"password": []byte("$3cr3t"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			res, err := eu.Create(ctx, test.cr)
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

func TestUserUpdate(t *testing.T) {
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
		expectedObservation managed.ExternalUpdate
		mock                func()
		errContains         string
	}{
		{
			scenario: "API ionoscloud returns an error",
			mock: func() {
				err := errors.New("internal error")
				client.EXPECT().UpdateUser(ctx, userIDInTest, gomock.Any()).Return(ionoscloud.User{}, nil, err)
			},
			cr: &v1alpha1.User{Status: v1alpha1.UserStatus{
				AtProvider: v1alpha1.UserObservation{
					UserID: userIDInTest,
				},
			}},
			errContains: "failed to update user",
		},
		{
			scenario: "User update with a group results in connection details",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusAccepted})
				user := ionoscloud.User{
					Id: ptr.To(userIDInTest),
					Properties: &ionoscloud.UserProperties{
						Email:             ptr.To("xplane-user@ionoscloud.io"),
						Firstname:         ptr.To("user name"),
						Lastname:          ptr.To("test"),
						S3CanonicalUserId: ptr.To("400c7ccfed0d"),
						Administrator:     ptr.To(false),
						ForceSecAuth:      ptr.To(false),
						SecAuthActive:     ptr.To(false),
						Active:            ptr.To(true),
					},
				}
				var p v1alpha1.UserParameters
				client.EXPECT().UpdateUser(ctx, userIDInTest, gomock.AssignableToTypeOf(p)).
					DoAndReturn(func(ctx context.Context, id string, p v1alpha1.UserParameters) (ionoscloud.User, *ionoscloud.APIResponse, error) {
						user.Properties.Email = &p.Email
						return user, apires, nil
					})
				client.EXPECT().AddUserToGroup(ctx, groupIDInTest, userIDInTest).Return(user, apires, nil)
			},
			cr: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					ForProvider: userParams(func(p *v1alpha1.UserParameters) {
						p.Password = "anotherpassw"
						p.Email = "anotheremail@ionoscloud.io"
					}),
				},
				Status: v1alpha1.UserStatus{
					AtProvider: v1alpha1.UserObservation{
						UserID: userIDInTest,
					},
				},
			},
			expectations: func(mg resource.Managed) {
				cr := mg.(*v1alpha1.User)
				g.Expect(cr.Spec.ForProvider.Password).To(BeEmpty())
				g.Expect(cr.Spec.ForProvider.Email).To(Equal("anotheremail@ionoscloud.io"))
			},
			expectedObservation: managed.ExternalUpdate{
				ConnectionDetails: managed.ConnectionDetails{
					"email":    []byte("anotheremail@ionoscloud.io"),
					"password": []byte("anotherpassw"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			res, err := eu.Update(ctx, test.cr)
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

func TestUserDelete(t *testing.T) {
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
		expectedObservation managed.ExternalCreation
		mock                func()
		errContains         string
	}{
		{
			scenario: "API delete user returns an error",
			mock: func() {
				err := errors.New("internal error")
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusInternalServerError})
				client.EXPECT().DeleteUser(ctx, gomock.Any()).Return(apires, err)
			},
			cr: &v1alpha1.User{Status: v1alpha1.UserStatus{
				AtProvider: v1alpha1.UserObservation{
					UserID: userIDInTest,
				},
			}},
			errContains: "failed to delete user",
		},
		{
			scenario: "API remove user from group returns an error",
			mock: func() {
				err := errors.New("internal error")
				client.EXPECT().DeleteUserFromGroup(ctx, groupIDInTest, userIDInTest).Return(err)
			},
			cr: &v1alpha1.User{Status: v1alpha1.UserStatus{
				AtProvider: v1alpha1.UserObservation{
					UserID:   userIDInTest,
					GroupIDs: []string{groupIDInTest},
				},
			}},
			errContains: "failed to remove user from a group",
		},
		{
			scenario: "User deleted successfully",
			mock: func() {
				apires := ionoscloud.NewAPIResponse(&http.Response{StatusCode: http.StatusAccepted})
				client.EXPECT().DeleteUserFromGroup(ctx, groupIDInTest, userIDInTest).Return(nil)
				client.EXPECT().DeleteUser(ctx, gomock.Any()).Return(apires, nil)
			},
			cr: &v1alpha1.User{Status: v1alpha1.UserStatus{
				AtProvider: v1alpha1.UserObservation{
					UserID:   userIDInTest,
					GroupIDs: []string{groupIDInTest},
				},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if test.mock != nil {
				test.mock()
			}
			err := eu.Delete(ctx, test.cr)
			if test.errContains != "" {
				g.Expect(err).ToNot(BeNil())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
				return
			}
			if test.expectations != nil {
				test.expectations(test.cr)
			}
			g.Expect(err).ToNot(HaveOccurred())
		})
	}
}

var defaultParams func(*v1alpha1.UserParameters) = nil

func userParams(mod func(*v1alpha1.UserParameters)) v1alpha1.UserParameters {
	p := &v1alpha1.UserParameters{
		Email:         "xplane-user@ionoscloud.io",
		FirstName:     "user name",
		LastName:      "test",
		Administrator: false,
		ForceSecAuth:  false,
		Password:      "$3cr3t",
		SecAuthActive: false,
		Active:        false,
		GroupIDs:      []string{"5458a703-6450-4ddd-b133-59349c83f832"},
	}
	if mod != nil {
		mod(p)
	}
	return *p
}
