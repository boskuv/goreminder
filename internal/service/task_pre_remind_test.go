package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/models"
)

func TestValidatePreRemindBeforeSeconds(t *testing.T) {
	assert.NoError(t, validatePreRemindBeforeSeconds(nil))
	zero := int64(0)
	assert.NoError(t, validatePreRemindBeforeSeconds(&zero))
	ok := int64(900)
	assert.NoError(t, validatePreRemindBeforeSeconds(&ok))
	neg := int64(-1)
	assert.Error(t, validatePreRemindBeforeSeconds(&neg))
	tooBig := maxPreRemindBeforeSeconds + 1
	assert.Error(t, validatePreRemindBeforeSeconds(&tooBig))
}

func TestApplyPreRemindBeforeSecondsUpdate(t *testing.T) {
	task := &models.Task{}
	sec := int64(600)
	require.NoError(t, applyPreRemindBeforeSecondsUpdate(task, &sec))
	require.NotNil(t, task.PreRemindBeforeSeconds)
	assert.Equal(t, int64(600), *task.PreRemindBeforeSeconds)

	zero := int64(0)
	require.NoError(t, applyPreRemindBeforeSecondsUpdate(task, &zero))
	assert.Nil(t, task.PreRemindBeforeSeconds)

	assert.NoError(t, applyPreRemindBeforeSecondsUpdate(task, nil))
}

func TestPreRemindBeforeSecondsEqual(t *testing.T) {
	a := int64(10)
	b := int64(10)
	c := int64(11)
	assert.True(t, preRemindBeforeSecondsEqual(nil, nil))
	assert.False(t, preRemindBeforeSecondsEqual(&a, nil))
	assert.True(t, preRemindBeforeSecondsEqual(&a, &b))
	assert.False(t, preRemindBeforeSecondsEqual(&a, &c))
}
