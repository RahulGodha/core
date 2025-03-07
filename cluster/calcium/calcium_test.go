package calcium

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	schedulermocks "github.com/projecteru2/core/scheduler/mocks"
	sourcemocks "github.com/projecteru2/core/source/mocks"
	storemocks "github.com/projecteru2/core/store/mocks"
	"github.com/projecteru2/core/types"
	"github.com/projecteru2/core/wal"
	walmocks "github.com/projecteru2/core/wal/mocks"
)

// DummyLock replace lock for testing
type dummyLock struct {
	m sync.Mutex
}

// Lock for lock
func (d *dummyLock) Lock(ctx context.Context) (context.Context, error) {
	d.m.Lock()
	return context.Background(), nil
}

// Unlock for unlock
func (d *dummyLock) Unlock(ctx context.Context) error {
	d.m.Unlock()
	return nil
}

func NewTestCluster() *Calcium {
	walDir, err := ioutil.TempDir(os.TempDir(), "core.wal.*")
	if err != nil {
		panic(err)
	}

	c := &Calcium{}
	c.config = types.Config{
		GlobalTimeout: 30 * time.Second,
		Git: types.GitConfig{
			CloneTimeout: 300 * time.Second,
		},
		Scheduler: types.SchedConfig{
			MaxShare:  -1,
			ShareBase: 100,
		},
		WALFile:        filepath.Join(walDir, "core.wal.log"),
		MaxConcurrency: 10,
	}
	c.store = &storemocks.Store{}
	c.scheduler = &schedulermocks.Scheduler{}
	c.source = &sourcemocks.Source{}
	c.wal = &WAL{WAL: &walmocks.WAL{}}

	mwal := c.wal.WAL.(*walmocks.WAL)
	commit := wal.Commit(func() error { return nil })
	mwal.On("Log", mock.Anything, mock.Anything).Return(commit, nil)

	return c
}

func TestNewCluster(t *testing.T) {
	config := types.Config{WALFile: "/tmp/a"}
	_, err := New(config, false)
	assert.Error(t, err)

	c, err := New(config, true)
	assert.NoError(t, err)

	c.Finalizer()
	privFile, err := ioutil.TempFile("", "priv")
	assert.NoError(t, err)
	_, err = privFile.WriteString("privkey")
	assert.NoError(t, err)
	defer privFile.Close()

	config.Git = types.GitConfig{PrivateKey: privFile.Name()}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		config.Git.SCMType = "gitlab"
		config.WALFile = "/tmp/b"
		c, err := New(config, true)
		assert.NoError(t, err)
		c.Finalizer()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		config.WALFile = "/tmp/c"
		config.Git.SCMType = "github"
		c, err := New(config, true)
		assert.NoError(t, err)
		c.Finalizer()
	}()

	wg.Wait()
}

func TestFinalizer(t *testing.T) {
	c := NewTestCluster()
	store := &storemocks.Store{}
	c.store = store
	store.On("TerminateEmbededStorage").Return(nil)
	c.Finalizer()
}
