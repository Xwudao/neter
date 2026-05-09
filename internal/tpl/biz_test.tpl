{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.MockTemplateData*/ -}}
package biz_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"{{.ModName}}/internal/biz/mocks"
)

// Test{{.StructBizName}}_Example is a generated test stub.
// Replace the TODO comments with real test cases.
func Test{{.StructBizName}}_Example(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockDep := mocks.NewMock{{.MockTypeName}}(ctrl)

	// 1. Set up expectations:
	//    mockDep.EXPECT().SomeMethod(gomock.Any()).Return(value, nil)
	//
	// 2. Create the biz (adapt constructor args if needed, e.g. jwt.Client):
	//    log := zap.NewNop().Sugar()
	//    appCtx := system.NewTestAppContext()
	//    b := biz.New{{.StructBizName}}(log, mockRepo, appCtx)
	//
	// 3. Call methods and assert results:
	//    result, err := b.SomeMethod(context.Background(), ...)
	//    assert.NoError(t, err)
	//    assert.NotNil(t, result)

	assert.NotNil(t, mockDep)
}
