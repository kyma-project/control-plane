package main

import (
	"fmt"
	"io/fs"
	"os"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/kyma-project/control-plane/components/schema-migrator/mocks"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func Test_copyFile(t *testing.T) {
	t.Run("Should return error while opening source file", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Open", "src").Return(nil, fmt.Errorf("failed to open file"))

		// when
		err := copyFile("src", "dst", mfs)

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while creating destination file", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(nil, fmt.Errorf("failed to create file"))

		// when
		err := copyFile("src", "dst", mfs)

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while copying file", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(0), fmt.Errorf("failed to copy file"))

		// when
		err := copyFile("src", "dst", mfs)

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while returning FileInfo", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfi := &mocks.MyFileInfo{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(65), nil)
		mfs.On("Stat", "src").Return(mfi, fmt.Errorf("failed to get FileInfo"))

		// when
		err := copyFile("src", "dst", mfs)

		// then
		assert.Error(t, err)
	})
	t.Run("Should return error while changing the mode of the file", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfi := &mocks.MyFileInfo{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(65), nil)
		mfs.On("Stat", "src").Return(mfi, nil)
		mfi.On("Mode").Return(fs.FileMode(0666))
		mfs.On("Chmod", "dst", fs.FileMode(0666)).Return(fmt.Errorf("failed to change file mode"))
		// when
		err := copyFile("src", "dst", mfs)

		// then
		assert.Error(t, err)
	})
	t.Run("Should copy the file", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfi := &mocks.MyFileInfo{}
		mfs.On("Open", "src").Return(&os.File{}, nil)
		mfs.On("Create", "dst").Return(&os.File{}, nil)
		mfs.On("Copy", &os.File{}, &os.File{}).Return(int64(65), nil)
		mfs.On("Stat", "src").Return(mfi, nil)
		mfi.On("Mode").Return(fs.FileMode(0666))
		mfs.On("Chmod", "dst", fs.FileMode(0666)).Return(nil)
		// when
		err := copyFile("src", "dst", mfs)

		// then
		assert.Nil(t, err)
	})

}

func Test_copyDir(t *testing.T) {
	t.Run("Should succesfully copy files", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("Create", "src").Return(os.Create("src"))
		mfs.On("ReadDir", "src").Return([]fs.DirEntry{}, nil)

		// when
		err := copyDir("src", "dst", mfs)

		// then
		assert.Nil(t, err)

	})
	t.Run("Should return error while reading directory", func(t *testing.T) {
		// given
		mfs := &mocks.FileSystem{}
		mfs.On("ReadDir", "src").Return(nil, fmt.Errorf("failed to read directory"))

		// when
		err := copyDir("src", "dst", mfs)

		// then
		assert.Error(t, err)

	})

}
