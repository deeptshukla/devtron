// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	app "github.com/devtron-labs/devtron/pkg/app/status"
	mock "github.com/stretchr/testify/mock"

	pg "github.com/go-pg/pg"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

// PipelineStatusTimelineResourcesService is an autogenerated mock type for the PipelineStatusTimelineResourcesService type
type PipelineStatusTimelineResourcesService struct {
	mock.Mock
}

// GetTimelineResourcesForATimeline provides a mock function with given fields: cdWfrId
func (_m *PipelineStatusTimelineResourcesService) GetTimelineResourcesForATimeline(cdWfrId int) ([]*app.SyncStageResourceDetailDto, error) {
	ret := _m.Called(cdWfrId)

	var r0 []*app.SyncStageResourceDetailDto
	if rf, ok := ret.Get(0).(func(int) []*app.SyncStageResourceDetailDto); ok {
		r0 = rf(cdWfrId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*app.SyncStageResourceDetailDto)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(cdWfrId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveOrUpdateCdPipelineTimelineResources provides a mock function with given fields: cdWfrId, application, tx, userId
func (_m *PipelineStatusTimelineResourcesService) SaveOrUpdateCdPipelineTimelineResources(cdWfrId int, application *v1alpha1.Application, tx *pg.Tx, userId int32) error {
	ret := _m.Called(cdWfrId, application, tx, userId)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, *v1alpha1.Application, *pg.Tx, int32) error); ok {
		r0 = rf(cdWfrId, application, tx, userId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewPipelineStatusTimelineResourcesService interface {
	mock.TestingT
	Cleanup(func())
}

// NewPipelineStatusTimelineResourcesService creates a new instance of PipelineStatusTimelineResourcesService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewPipelineStatusTimelineResourcesService(t mockConstructorTestingTNewPipelineStatusTimelineResourcesService) *PipelineStatusTimelineResourcesService {
	mock := &PipelineStatusTimelineResourcesService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
