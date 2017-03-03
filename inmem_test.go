package inmem_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/facebookgo/ensure"
	"github.com/magiccarpetsoftware/inmem"
)

func emptyString(t *testing.T, s string) {
	if s != "" {
		fmt.Printf("string not empty: %v", s)
		t.Fail()
	}
}

func testManyThings(t *testing.T, c inmem.Cache) {
	const (
		k  = "appts"
		v1 = "https://newtest-appts.something-something.ops-is-awesome.com.au"
		v2 = "https://prod-appts.something-something.ops-is-awesome.com.au"
	)

	// it's empty
	ensure.DeepEqual(t, c.Len(), 0)

	// not there to start with
	actual, found := c.Get(k)
	ensure.False(t, found)
	emptyString(t, actual)

	// add it
	c.Add(k, v1, time.Now().Add(time.Hour))

	// now it's there
	actual, found = c.Get(k)
	ensure.True(t, found)
	ensure.DeepEqual(t, actual, v1)

	// we have some items
	ensure.DeepEqual(t, c.Len(), 1)

	// replace it
	c.Add(k, v2, time.Now().Add(time.Hour))

	// now find the new value
	actual, found = c.Get(k)
	ensure.True(t, found)
	ensure.DeepEqual(t, actual, v2)

	// we still only have 1 item
	ensure.DeepEqual(t, c.Len(), 1)

	// remove it
	c.Remove(k)

	// not there any more
	actual, found = c.Get(k)
	ensure.False(t, found)
	emptyString(t, actual)

	// it's empty again
	ensure.DeepEqual(t, c.Len(), 0)
}

func TestManyThingsUnlocked(t *testing.T) {
	testManyThings(t, inmem.NewUnlocked(10))
}

func TestManyThingsLocked(t *testing.T) {
	c, err := inmem.NewLockedString(10)
	if err != nil {
		t.Errorf("error initializing locked string cache: %v", err)
	}
	testManyThings(t, c)
}

func TestPanicNewUnlockedSizeZero(t *testing.T) {
	defer ensure.PanicDeepEqual(t, "cache: must provide a positive size")
	_ = inmem.NewUnlocked(0)
}

func TestPanicNewLockedSizeZero(t *testing.T) {
	_, err := inmem.NewLockedString(0)
	if err == nil {
		t.Fail()
	}
}

func TestCacheSize(t *testing.T) {
	c := inmem.NewUnlocked(2)
	e := time.Now().Add(time.Hour)
	c.Add("1", "1", e)
	c.Add("2", "2", e)
	c.Add("3", "3", e)
	ensure.DeepEqual(t, c.Len(), 2)
	_, found := c.Get("1")
	ensure.False(t, found)
}

func TestTTLExpired(t *testing.T) {
	c := inmem.NewUnlocked(2)
	e := time.Now().Add(-time.Hour)
	c.Add("1", "1", e)
	ensure.DeepEqual(t, c.Len(), 1)
	_, found := c.Get("1")
	ensure.False(t, found)
	ensure.DeepEqual(t, c.Len(), 0)
}
