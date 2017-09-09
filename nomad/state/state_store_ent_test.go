// +build ent

package state

import (
	"sort"
	"testing"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/stretchr/testify/assert"
)

func TestStateStore_UpsertSentinelPolicy(t *testing.T) {
	state := testStateStore(t)
	policy := mock.SentinelPolicy()
	policy2 := mock.SentinelPolicy()

	ws := memdb.NewWatchSet()
	if _, err := state.SentinelPolicyByName(ws, policy.Name); err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err := state.SentinelPolicyByName(ws, policy2.Name); err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := state.UpsertSentinelPolicies(1000,
		[]*structs.SentinelPolicy{policy, policy2}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !watchFired(ws) {
		t.Fatalf("bad")
	}

	ws = memdb.NewWatchSet()
	out, err := state.SentinelPolicyByName(ws, policy.Name)
	assert.Equal(t, nil, err)
	assert.Equal(t, policy, out)

	out, err = state.SentinelPolicyByName(ws, policy2.Name)
	assert.Equal(t, nil, err)
	assert.Equal(t, policy2, out)

	iter, err := state.SentinelPolicies(ws)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we see both policies
	count := 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	if count != 2 {
		t.Fatalf("bad: %d", count)
	}

	iter, err = state.SentinelPoliciesByScope(ws, "submit-job")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we see both policies
	count = 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	if count != 2 {
		t.Fatalf("bad: %d", count)
	}

	index, err := state.Index("sentinel_policy")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}

	if watchFired(ws) {
		t.Fatalf("bad")
	}
}

func TestStateStore_DeleteSentinelPolicy(t *testing.T) {
	state := testStateStore(t)
	policy := mock.SentinelPolicy()
	policy2 := mock.SentinelPolicy()

	// Create the policy
	if err := state.UpsertSentinelPolicies(1000,
		[]*structs.SentinelPolicy{policy, policy2}); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a watcher
	ws := memdb.NewWatchSet()
	if _, err := state.SentinelPolicyByName(ws, policy.Name); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Delete the policy
	if err := state.DeleteSentinelPolicies(1001,
		[]string{policy.Name, policy2.Name}); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure watching triggered
	if !watchFired(ws) {
		t.Fatalf("bad")
	}

	// Ensure we don't get the object back
	ws = memdb.NewWatchSet()
	out, err := state.SentinelPolicyByName(ws, policy.Name)
	assert.Equal(t, nil, err)
	if out != nil {
		t.Fatalf("bad: %#v", out)
	}

	iter, err := state.SentinelPolicies(ws)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we see both policies
	count := 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	if count != 0 {
		t.Fatalf("bad: %d", count)
	}

	index, err := state.Index("sentinel_policy")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	if watchFired(ws) {
		t.Fatalf("bad")
	}
}

func TestStateStore_SentinelPolicyByNamePrefix(t *testing.T) {
	state := testStateStore(t)
	names := []string{
		"foo",
		"bar",
		"foobar",
		"foozip",
		"zip",
	}

	// Create the policies
	var baseIndex uint64 = 1000
	for _, name := range names {
		p := mock.SentinelPolicy()
		p.Name = name
		if err := state.UpsertSentinelPolicies(baseIndex, []*structs.SentinelPolicy{p}); err != nil {
			t.Fatalf("err: %v", err)
		}
		baseIndex++
	}

	// Scan by prefix
	iter, err := state.SentinelPolicyByNamePrefix(nil, "foo")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure we see both policies
	count := 0
	out := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
		out = append(out, raw.(*structs.SentinelPolicy).Name)
	}
	if count != 3 {
		t.Fatalf("bad: %d %v", count, out)
	}
	sort.Strings(out)

	expect := []string{"foo", "foobar", "foozip"}
	assert.Equal(t, expect, out)
}

func TestStateStore_RestoreSentinelPolicy(t *testing.T) {
	state := testStateStore(t)
	policy := mock.SentinelPolicy()

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.SentinelPolicyRestore(policy)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	restore.Commit()

	ws := memdb.NewWatchSet()
	out, err := state.SentinelPolicyByName(ws, policy.Name)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	assert.Equal(t, policy, out)
}

func TestStateStore_UpsertQuotaSpec(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	qs1 := mock.QuotaSpec()
	qs2 := mock.QuotaSpec()

	ws := memdb.NewWatchSet()
	assert.Nil(state.QuotaSpecByName(ws, qs1.Name))
	assert.Nil(state.QuotaSpecByName(ws, qs2.Name))

	assert.Nil(state.UpsertQuotaSpecs(1000, []*structs.QuotaSpec{qs1, qs2}))
	assert.True(watchFired(ws))

	ws = memdb.NewWatchSet()
	out, err := state.QuotaSpecByName(ws, qs1.Name)
	assert.Nil(err)
	assert.Equal(qs1, out)

	out, err = state.QuotaSpecByName(ws, qs2.Name)
	assert.Nil(err)
	assert.Equal(qs2, out)

	iter, err := state.QuotaSpecs(ws)
	assert.Nil(err)

	// Ensure we see both specs
	count := 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	assert.Equal(2, count)

	index, err := state.Index(TableQuotaSpec)
	assert.Nil(err)
	assert.EqualValues(1000, index)
	assert.False(watchFired(ws))
}

func TestStateStore_DeleteQuotaSpecs(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	qs1 := mock.QuotaSpec()
	qs2 := mock.QuotaSpec()

	// Create the quota specs
	assert.Nil(state.UpsertQuotaSpecs(1000, []*structs.QuotaSpec{qs1, qs2}))

	// Create a watcher
	ws := memdb.NewWatchSet()
	_, err := state.QuotaSpecByName(ws, qs1.Name)
	assert.Nil(err)

	// Delete the spec
	assert.Nil(state.DeleteQuotaSpecs(1001, []string{qs1.Name, qs2.Name}))

	// Ensure watching triggered
	assert.True(watchFired(ws))

	// Ensure we don't get the object back
	ws = memdb.NewWatchSet()
	out, err := state.QuotaSpecByName(ws, qs1.Name)
	assert.Nil(err)
	assert.Nil(out)

	iter, err := state.QuotaSpecs(ws)
	assert.Nil(err)

	// Ensure we see both policies
	count := 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	assert.Zero(count)

	index, err := state.Index(TableQuotaSpec)
	assert.Nil(err)
	assert.EqualValues(1001, index)
	assert.False(watchFired(ws))
}

func TestStateStore_QuotaSpecsByNamePrefix(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	names := []string{
		"foo",
		"bar",
		"foobar",
		"foozip",
		"zip",
	}

	// Create the policies
	var baseIndex uint64 = 1000
	for _, name := range names {
		qs := mock.QuotaSpec()
		qs.Name = name
		assert.Nil(state.UpsertQuotaSpecs(baseIndex, []*structs.QuotaSpec{qs}))
		baseIndex++
	}

	// Scan by prefix
	iter, err := state.QuotaSpecByNamePrefix(nil, "foo")
	assert.Nil(err)

	// Ensure we see both policies
	count := 0
	out := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
		out = append(out, raw.(*structs.QuotaSpec).Name)
	}
	assert.Equal(3, count)
	sort.Strings(out)

	expect := []string{"foo", "foobar", "foozip"}
	assert.Equal(expect, out)
}

func TestStateStore_RestoreQuotaSpec(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	spec := mock.QuotaSpec()

	restore, err := state.Restore()
	assert.Nil(err)

	err = restore.QuotaSpecRestore(spec)
	assert.Nil(err)
	restore.Commit()

	ws := memdb.NewWatchSet()
	out, err := state.QuotaSpecByName(ws, spec.Name)
	assert.Nil(err)
	assert.Equal(spec, out)
}

func TestStateStore_UpsertQuotaUsage(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	qu1 := mock.QuotaUsage()
	qu2 := mock.QuotaUsage()

	ws := memdb.NewWatchSet()
	assert.Nil(state.QuotaUsageByName(ws, qu1.Name))
	assert.Nil(state.QuotaUsageByName(ws, qu2.Name))

	assert.Nil(state.UpsertQuotaUsages(1000, []*structs.QuotaUsage{qu1, qu2}))
	assert.True(watchFired(ws))

	ws = memdb.NewWatchSet()
	out, err := state.QuotaUsageByName(ws, qu1.Name)
	assert.Nil(err)
	assert.Equal(qu1, out)

	out, err = state.QuotaUsageByName(ws, qu2.Name)
	assert.Nil(err)
	assert.Equal(qu2, out)

	iter, err := state.QuotaUsages(ws)
	assert.Nil(err)

	// Ensure we see both usages
	count := 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	assert.Equal(2, count)

	index, err := state.Index(TableQuotaUsage)
	assert.Nil(err)
	assert.EqualValues(1000, index)
	assert.False(watchFired(ws))
}

func TestStateStore_DeleteQuotaUsages(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	qu1 := mock.QuotaUsage()
	qu2 := mock.QuotaUsage()

	// Create the quota usages
	assert.Nil(state.UpsertQuotaUsages(1000, []*structs.QuotaUsage{qu1, qu2}))

	// Create a watcher
	ws := memdb.NewWatchSet()
	_, err := state.QuotaUsageByName(ws, qu1.Name)
	assert.Nil(err)

	// Delete the usage
	assert.Nil(state.DeleteQuotaUsages(1001, []string{qu1.Name, qu2.Name}))

	// Ensure watching triggered
	assert.True(watchFired(ws))

	// Ensure we don't get the object back
	ws = memdb.NewWatchSet()
	out, err := state.QuotaUsageByName(ws, qu1.Name)
	assert.Nil(err)
	assert.Nil(out)

	iter, err := state.QuotaUsages(ws)
	assert.Nil(err)

	// Ensure we see both policies
	count := 0
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
	}
	assert.Zero(count)

	index, err := state.Index(TableQuotaUsage)
	assert.Nil(err)
	assert.EqualValues(1001, index)
	assert.False(watchFired(ws))
}

func TestStateStore_QuotaUsagesByNamePrefix(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	names := []string{
		"foo",
		"bar",
		"foobar",
		"foozip",
		"zip",
	}

	// Create the policies
	var baseIndex uint64 = 1000
	for _, name := range names {
		qu := mock.QuotaUsage()
		qu.Name = name
		assert.Nil(state.UpsertQuotaUsages(baseIndex, []*structs.QuotaUsage{qu}))
		baseIndex++
	}

	// Scan by prefix
	iter, err := state.QuotaUsageByNamePrefix(nil, "foo")
	assert.Nil(err)

	// Ensure we see both policies
	count := 0
	out := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		count++
		out = append(out, raw.(*structs.QuotaUsage).Name)
	}
	assert.Equal(3, count)
	sort.Strings(out)

	expect := []string{"foo", "foobar", "foozip"}
	assert.Equal(expect, out)
}

func TestStateStore_RestoreQuotaUsage(t *testing.T) {
	assert := assert.New(t)
	state := testStateStore(t)
	usage := mock.QuotaUsage()

	restore, err := state.Restore()
	assert.Nil(err)

	err = restore.QuotaUsageRestore(usage)
	assert.Nil(err)
	restore.Commit()

	ws := memdb.NewWatchSet()
	out, err := state.QuotaUsageByName(ws, usage.Name)
	assert.Nil(err)
	assert.Equal(usage, out)
}
