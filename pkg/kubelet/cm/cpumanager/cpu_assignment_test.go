/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cpumanager

import (
	"reflect"
	"sort"
	"testing"

	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/test/utils/ktesting"
	"k8s.io/utils/cpuset"
)

func TestCPUAccumulatorFreeSockets(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description   string
		topo          *topology.CPUTopology
		availableCPUs cpuset.CPUSet
		expect        []int
	}{
		{
			"single socket HT, 1 socket free",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]int{0},
		},
		{
			"single socket HT, 0 sockets free",
			topoSingleSocketHT,
			cpuset.New(1, 2, 3, 4, 5, 6, 7),
			[]int{},
		},
		{
			"dual socket HT, 2 sockets free",
			topoDualSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			[]int{0, 1},
		},
		{
			"dual socket HT, 1 socket free",
			topoDualSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11),
			[]int{1},
		},
		{
			"dual socket HT, 0 sockets free",
			topoDualSocketHT,
			cpuset.New(0, 2, 3, 4, 5, 6, 7, 8, 9, 11),
			[]int{},
		},
		{
			"dual socket, multi numa per socket, HT, 2 sockets free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			[]int{0, 1},
		},
		{
			"dual socket, multi numa per socket, HT, 1 sockets free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-79"),
			[]int{1},
		},
		{
			"dual socket, multi numa per socket, HT, 0 sockets free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-78"),
			[]int{},
		},
		{
			"dual numa, multi socket per per socket, HT, 4 sockets free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-79"),
			[]int{0, 1, 2, 3},
		},
		{
			"dual numa, multi socket per per socket, HT, 3 sockets free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-19,21-79"),
			[]int{0, 1, 3},
		},
		{
			"dual numa, multi socket per per socket, HT, 2 sockets free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-59,61-78"),
			[]int{0, 1},
		},
		{
			"dual numa, multi socket per per socket, HT, 1 sockets free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "1-19,21-38,41-60,61-78"),
			[]int{1},
		},
		{
			"dual numa, multi socket per per socket, HT, 0 sockets free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-40,42-49,51-68,71-79"),
			[]int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			acc := newCPUAccumulator(logger, tc.topo, tc.availableCPUs, 0, CPUSortingStrategyPacked)
			result := acc.freeSockets()
			sort.Ints(result)
			if !reflect.DeepEqual(result, tc.expect) {
				t.Errorf("expected %v to equal %v", result, tc.expect)

			}
		})
	}
}

func TestCPUAccumulatorFreeNUMANodes(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description   string
		topo          *topology.CPUTopology
		availableCPUs cpuset.CPUSet
		expect        []int
	}{
		{
			"single socket HT, 1 NUMA node free",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]int{0},
		},
		{
			"single socket HT, 0 NUMA Node free",
			topoSingleSocketHT,
			cpuset.New(1, 2, 3, 4, 5, 6, 7),
			[]int{},
		},
		{
			"dual socket HT, 2 NUMA Node free",
			topoDualSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			[]int{0, 1},
		},
		{
			"dual socket HT, 1 NUMA Node free",
			topoDualSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11),
			[]int{1},
		},
		{
			"dual socket HT, 0 NUMA node free",
			topoDualSocketHT,
			cpuset.New(0, 2, 3, 4, 5, 6, 7, 8, 9, 11),
			[]int{},
		},
		{
			"dual socket, multi numa per socket, HT, 4 NUMA Node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			[]int{0, 1, 2, 3},
		},
		{
			"dual socket, multi numa per socket, HT, 3 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-79"),
			[]int{1, 2, 3},
		},
		{
			"dual socket, multi numa per socket, HT, 2 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-9,11-79"),
			[]int{2, 3},
		},
		{
			"dual socket, multi numa per socket, HT, 1 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-9,11-59,61-79"),
			[]int{3},
		},
		{
			"dual socket, multi numa per socket, HT, 0 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-9,11-59,61-78"),
			[]int{},
		},
		{
			"dual numa, multi socket per per socket, HT, 2 NUMA node free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-79"),
			[]int{0, 1},
		},
		{
			"dual numa, multi socket per per socket, HT, 1 NUMA node free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-9,11-79"),
			[]int{1},
		},
		{
			"dual numa, multi socket per per socket, HT, 0 sockets free",
			fakeTopoMultiSocketDualSocketPerNumaHT,
			mustParseCPUSet(t, "0-9,11-59,61-79"),
			[]int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			acc := newCPUAccumulator(logger, tc.topo, tc.availableCPUs, 0, CPUSortingStrategyPacked)
			result := acc.freeNUMANodes()
			if !reflect.DeepEqual(result, tc.expect) {
				t.Errorf("expected %v to equal %v", result, tc.expect)
			}
		})
	}
}

func TestCPUAccumulatorFreeSocketsAndNUMANodes(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description     string
		topo            *topology.CPUTopology
		availableCPUs   cpuset.CPUSet
		expectSockets   []int
		expectNUMANodes []int
	}{
		{
			"dual socket, multi numa per socket, HT, 2 Socket/4 NUMA Node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			[]int{0, 1},
			[]int{0, 1, 2, 3},
		},
		{
			"dual socket, multi numa per socket, HT, 1 Socket/3 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-79"),
			[]int{1},
			[]int{1, 2, 3},
		},
		{
			"dual socket, multi numa per socket, HT, 1 Socket/ 2 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-9,11-79"),
			[]int{1},
			[]int{2, 3},
		},
		{
			"dual socket, multi numa per socket, HT, 0 Socket/ 2 NUMA node free",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-59,61-79"),
			[]int{},
			[]int{1, 3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			acc := newCPUAccumulator(logger, tc.topo, tc.availableCPUs, 0, CPUSortingStrategyPacked)
			resultNUMANodes := acc.freeNUMANodes()
			if !reflect.DeepEqual(resultNUMANodes, tc.expectNUMANodes) {
				t.Errorf("expected NUMA Nodes %v to equal %v", resultNUMANodes, tc.expectNUMANodes)
			}
			resultSockets := acc.freeSockets()
			if !reflect.DeepEqual(resultSockets, tc.expectSockets) {
				t.Errorf("expected Sockets %v to equal %v", resultSockets, tc.expectSockets)
			}
		})
	}
}

func TestCPUAccumulatorFreeCores(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description   string
		topo          *topology.CPUTopology
		availableCPUs cpuset.CPUSet
		expect        []int
	}{
		{
			"single socket HT, 4 cores free",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]int{0, 1, 2, 3},
		},
		{
			"single socket HT, 3 cores free",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 4, 5, 6),
			[]int{0, 1, 2},
		},
		{
			"single socket HT, 3 cores free (1 partially consumed)",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6),
			[]int{0, 1, 2},
		},
		{
			"single socket HT, 0 cores free",
			topoSingleSocketHT,
			cpuset.New(),
			[]int{},
		},
		{
			"single socket HT, 0 cores free (4 partially consumed)",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3),
			[]int{},
		},
		{
			"dual socket HT, 6 cores free",
			topoDualSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			[]int{0, 2, 4, 1, 3, 5},
		},
		{
			"dual socket HT, 5 cores free (1 consumed from socket 0)",
			topoDualSocketHT,
			cpuset.New(2, 1, 3, 4, 5, 7, 8, 9, 10, 11),
			[]int{2, 4, 1, 3, 5},
		},
		{
			"dual socket HT, 4 cores free (1 consumed from each socket)",
			topoDualSocketHT,
			cpuset.New(2, 3, 4, 5, 8, 9, 10, 11),
			[]int{2, 4, 3, 5},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			acc := newCPUAccumulator(logger, tc.topo, tc.availableCPUs, 0, CPUSortingStrategyPacked)
			result := acc.freeCores()
			if !reflect.DeepEqual(result, tc.expect) {
				t.Errorf("expected %v to equal %v", result, tc.expect)
			}
		})
	}
}

func TestCPUAccumulatorFreeCPUs(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description   string
		topo          *topology.CPUTopology
		availableCPUs cpuset.CPUSet
		expect        []int
	}{
		{
			"single socket HT, 8 cpus free",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]int{0, 4, 1, 5, 2, 6, 3, 7},
		},
		{
			"single socket HT, 5 cpus free",
			topoSingleSocketHT,
			cpuset.New(3, 4, 5, 6, 7),
			[]int{4, 5, 6, 3, 7},
		},
		{
			"dual socket HT, 12 cpus free",
			topoDualSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			[]int{0, 6, 2, 8, 4, 10, 1, 7, 3, 9, 5, 11},
		},
		{
			"dual socket HT, 11 cpus free",
			topoDualSocketHT,
			cpuset.New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			[]int{6, 2, 8, 4, 10, 1, 7, 3, 9, 5, 11},
		},
		{
			"dual socket HT, 10 cpus free",
			topoDualSocketHT,
			cpuset.New(1, 2, 3, 4, 5, 7, 8, 9, 10, 11),
			[]int{2, 8, 4, 10, 1, 7, 3, 9, 5, 11},
		},
		{
			"triple socket HT, 12 cpus free",
			topoTripleSocketHT,
			cpuset.New(0, 1, 2, 3, 6, 7, 8, 9, 10, 11, 12, 13),
			[]int{12, 13, 0, 1, 2, 3, 6, 7, 8, 9, 10, 11},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			acc := newCPUAccumulator(logger, tc.topo, tc.availableCPUs, 0, CPUSortingStrategyPacked)
			result := acc.freeCPUs()
			if !reflect.DeepEqual(result, tc.expect) {
				t.Errorf("expected %v to equal %v", result, tc.expect)
			}
		})
	}
}

func TestCPUAccumulatorTake(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description     string
		topo            *topology.CPUTopology
		availableCPUs   cpuset.CPUSet
		takeCPUs        []cpuset.CPUSet
		numCPUs         int
		expectSatisfied bool
		expectFailed    bool
	}{
		{
			"take 0 cpus from a single socket HT, require 1",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]cpuset.CPUSet{cpuset.New()},
			1,
			false,
			false,
		},
		{
			"take 0 cpus from a single socket HT, require 1, none available",
			topoSingleSocketHT,
			cpuset.New(),
			[]cpuset.CPUSet{cpuset.New()},
			1,
			false,
			true,
		},
		{
			"take 1 cpu from a single socket HT, require 1",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]cpuset.CPUSet{cpuset.New(0)},
			1,
			true,
			false,
		},
		{
			"take 1 cpu from a single socket HT, require 2",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]cpuset.CPUSet{cpuset.New(0)},
			2,
			false,
			false,
		},
		{
			"take 2 cpu from a single socket HT, require 4, expect failed",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2),
			[]cpuset.CPUSet{cpuset.New(0), cpuset.New(1)},
			4,
			false,
			true,
		},
		{
			"take all cpus one at a time from a single socket HT, require 8",
			topoSingleSocketHT,
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			[]cpuset.CPUSet{
				cpuset.New(0),
				cpuset.New(1),
				cpuset.New(2),
				cpuset.New(3),
				cpuset.New(4),
				cpuset.New(5),
				cpuset.New(6),
				cpuset.New(7),
			},
			8,
			true,
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			acc := newCPUAccumulator(logger, tc.topo, tc.availableCPUs, tc.numCPUs, CPUSortingStrategyPacked)
			totalTaken := 0
			for _, cpus := range tc.takeCPUs {
				acc.take(cpus)
				totalTaken += cpus.Size()
			}
			if tc.expectSatisfied != acc.isSatisfied() {
				t.Errorf("expected acc.isSatisfied() to be %t", tc.expectSatisfied)
			}
			if tc.expectFailed != acc.isFailed() {
				t.Errorf("expected acc.isFailed() to be %t", tc.expectFailed)
			}
			for _, cpus := range tc.takeCPUs {
				availableCPUs := acc.details.CPUs()
				if cpus.Intersection(availableCPUs).Size() > 0 {
					t.Errorf("expected intersection of taken cpus [%s] and acc.details.CPUs() [%s] to be empty", cpus, availableCPUs)
				}
				if !cpus.IsSubsetOf(acc.result) {
					t.Errorf("expected [%s] to be a subset of acc.result [%s]", cpus, acc.result)
				}
			}
			expNumCPUsNeeded := tc.numCPUs - totalTaken
			if acc.numCPUsNeeded != expNumCPUsNeeded {
				t.Errorf("expected acc.numCPUsNeeded to be %d (got %d)", expNumCPUsNeeded, acc.numCPUsNeeded)
			}
		})
	}
}

type takeByTopologyTestCase struct {
	description   string
	topo          *topology.CPUTopology
	opts          StaticPolicyOptions
	availableCPUs cpuset.CPUSet
	numCPUs       int
	expErr        string
	expResult     cpuset.CPUSet
}

func commonTakeByTopologyTestCases(t *testing.T) []takeByTopologyTestCase {
	return []takeByTopologyTestCase{
		{
			"take more cpus than are available from single socket with HT",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 2, 4, 6),
			5,
			"not enough cpus available to satisfy request: requested=5, available=4",
			cpuset.New(),
		},
		{
			"take zero cpus from single socket with HT",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			0,
			"",
			cpuset.New(),
		},
		{
			"take one cpu from single socket with HT",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			1,
			"",
			cpuset.New(0),
		},
		{
			"take one cpu from single socket with HT, some cpus are taken",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(1, 3, 5, 6, 7),
			1,
			"",
			cpuset.New(6),
		},
		{
			"take two cpus from single socket with HT",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			2,
			"",
			cpuset.New(0, 4),
		},
		{
			"take all cpus from single socket with HT",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			8,
			"",
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
		},
		{
			"take two cpus from single socket with HT, only one core totally free",
			topoSingleSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 1, 2, 3, 6),
			2,
			"",
			cpuset.New(2, 6),
		},
		{
			"take a socket of cpus from dual socket with HT",
			topoDualSocketHT,
			StaticPolicyOptions{},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			6,
			"",
			cpuset.New(0, 2, 4, 6, 8, 10),
		},
		{
			"take a socket of cpus from dual socket with multi-numa-per-socket with HT",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "0-79"),
			40,
			"",
			mustParseCPUSet(t, "0-19,40-59"),
		},
		{
			"take a NUMA node of cpus from dual socket with multi-numa-per-socket with HT",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "0-79"),
			20,
			"",
			mustParseCPUSet(t, "0-9,40-49"),
		},
		{
			"take a NUMA node of cpus from dual socket with multi-numa-per-socket with HT, with 1 NUMA node already taken",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "10-39,50-79"),
			20,
			"",
			mustParseCPUSet(t, "10-19,50-59"),
		},
		{
			"take a socket and a NUMA node of cpus from dual socket with multi-numa-per-socket with HT",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "0-79"),
			60,
			"",
			mustParseCPUSet(t, "0-29,40-69"),
		},
		{
			"take a socket and a NUMA node of cpus from dual socket with multi-numa-per-socket with HT, a core taken",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "1-39,41-79"), // reserve the first (phys) core (0,40)
			60,
			"",
			mustParseCPUSet(t, "10-39,50-79"),
		},
	}
}

func TestTakeByTopologyNUMAPacked(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := commonTakeByTopologyTestCases(t)
	testCases = append(testCases, []takeByTopologyTestCase{
		{
			"take one cpu from dual socket with HT - core from Socket 0",
			topoDualSocketHT,
			StaticPolicyOptions{},
			cpuset.New(1, 2, 3, 4, 5, 7, 8, 9, 10, 11),
			1,
			"",
			cpuset.New(2),
		},
		{
			"allocate 4 full cores with 3 coming from the first NUMA node (filling it up) and 1 coming from the second NUMA node",
			topoDualSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "0-11"),
			8,
			"",
			mustParseCPUSet(t, "0,6,2,8,4,10,1,7"),
		},
		{
			"allocate 32 full cores with 30 coming from the first 3 NUMA nodes (filling them up) and 2 coming from the fourth NUMA node",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{},
			mustParseCPUSet(t, "0-79"),
			64,
			"",
			mustParseCPUSet(t, "0-29,40-69,30,31,70,71"),
		},
		// Test cases for PreferAlignByUncoreCache
		{
			"take cpus from two full UncoreCaches and partial from a single UncoreCache",
			topoUncoreSingleSocketNoSMT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "1-15"),
			10,
			"",
			cpuset.New(1, 2, 4, 5, 6, 7, 8, 9, 10, 11),
		},
		{
			"take one cpu from dual socket with HT - core from Socket 0",
			topoDualSocketHT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			cpuset.New(1, 2, 3, 4, 5, 7, 8, 9, 10, 11),
			1,
			"",
			cpuset.New(1),
		},
		{
			"take first available UncoreCache from first socket",
			topoUncoreDualSocketNoSMT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "0-15"),
			4,
			"",
			cpuset.New(0, 1, 2, 3),
		},
		{
			"take all available UncoreCache from first socket",
			topoUncoreDualSocketNoSMT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "2-15"),
			6,
			"",
			cpuset.New(2, 3, 4, 5, 6, 7),
		},
		{
			"take first available UncoreCache from second socket",
			topoUncoreDualSocketNoSMT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "8-15"),
			4,
			"",
			cpuset.New(8, 9, 10, 11),
		},
		{
			"take first available UncoreCache from available NUMA",
			topoUncoreSingleSocketMultiNuma,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "3,4-8,12"),
			2,
			"",
			cpuset.New(4, 5),
		},
		{
			"take cpus from best available UncoreCache group of multi uncore cache single socket - SMT enabled",
			topoUncoreSingleSocketSMT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "2-3,10-11,4-7,12-15"),
			6,
			"",
			cpuset.New(4, 5, 6, 12, 13, 14),
		},
		{
			"take cpus from multiple UncoreCache of single socket - SMT enabled",
			topoUncoreSingleSocketSMT,
			StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			mustParseCPUSet(t, "1-7,9-15"),
			10,
			"",
			mustParseCPUSet(t, "4-7,12-15,1,9"),
		},
	}...)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			strategy := CPUSortingStrategyPacked
			if tc.opts.DistributeCPUsAcrossCores {
				strategy = CPUSortingStrategySpread
			}

			result, err := takeByTopologyNUMAPacked(logger, tc.topo, tc.availableCPUs, tc.numCPUs, strategy, tc.opts.PreferAlignByUncoreCacheOption)
			if tc.expErr != "" && err != nil && err.Error() != tc.expErr {
				t.Errorf("expected error to be [%v] but it was [%v]", tc.expErr, err)
			}
			if !result.Equals(tc.expResult) {
				t.Errorf("expected result [%s] to equal [%s]", result, tc.expResult)
			}
		})
	}
}

func TestTakeByTopologyWithSpreadPhysicalCPUsPreferredOption(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := []struct {
		description   string
		topo          *topology.CPUTopology
		opts          StaticPolicyOptions
		availableCPUs cpuset.CPUSet
		numCPUs       int
		expErr        string
		expResult     cpuset.CPUSet
	}{
		{
			"take a socket of cpus from single socket with HT, 3 cpus",
			topoSingleSocketHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			3,
			"",
			cpuset.New(0, 1, 2),
		},
		{
			"take a socket of cpus from dual socket with HT, 2 cpus",
			topoDualSocketHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			2,
			"",
			cpuset.New(0, 2),
		},
		{
			"take a socket of cpus from dual socket with HT, 3 cpus",
			topoDualSocketHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			3,
			"",
			cpuset.New(0, 2, 4),
		},
		{
			"take a socket of cpus from dual socket with HT, 6 cpus",
			topoDualSocketHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			6,
			"",
			cpuset.New(0, 2, 4, 6, 8, 10),
		},
		{
			"take cpus from dual socket with HT, 8 cpus",
			topoDualSocketHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			8,
			"",
			cpuset.New(0, 2, 4, 6, 8, 10, 1, 3),
		},
		{
			"take a socket of cpus from dual socket without HT, 2 cpus",
			topoDualSocketNoHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			cpuset.New(0, 1, 2, 3, 4, 5, 6, 7),
			2,
			"",
			cpuset.New(0, 1),
		},
		{
			// DistributeCPUsAcrossCores doesn't care socket and numa ranking. such setting in test is transparent.
			"take a socket of cpus from dual socket with multi numa per socket and HT, 12 cpus",
			topoDualSocketMultiNumaPerSocketHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			mustParseCPUSet(t, "0-79"),
			8,
			"",
			mustParseCPUSet(t, "0-7"),
		},
		{
			"take a socket of cpus from quad socket four way with HT, 12 cpus",
			topoQuadSocketFourWayHT,
			StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			mustParseCPUSet(t, "0-287"),
			12,
			"",
			mustParseCPUSet(t, "0-2,9-10,13-14,21-22,25-26,33"),
		},
	}

	for _, tc := range testCases {
		strategy := CPUSortingStrategyPacked
		if tc.opts.DistributeCPUsAcrossCores {
			strategy = CPUSortingStrategySpread
		}
		result, err := takeByTopologyNUMAPacked(logger, tc.topo, tc.availableCPUs, tc.numCPUs, strategy, tc.opts.PreferAlignByUncoreCacheOption)
		if tc.expErr != "" && err.Error() != tc.expErr {
			t.Errorf("testCase %q failed, expected error to be [%v] but it was [%v]", tc.description, tc.expErr, err)
		}
		if !result.Equals(tc.expResult) {
			t.Errorf("testCase %q failed, expected result [%s] to equal [%s]", tc.description, result, tc.expResult)
		}
	}
}

type takeByTopologyExtendedTestCase struct {
	description   string
	topo          *topology.CPUTopology
	availableCPUs cpuset.CPUSet
	numCPUs       int
	cpuGroupSize  int
	expErr        string
	expResult     cpuset.CPUSet
}

func commonTakeByTopologyExtendedTestCases(t *testing.T) []takeByTopologyExtendedTestCase {
	var extendedTestCases []takeByTopologyExtendedTestCase

	testCases := commonTakeByTopologyTestCases(t)
	for _, tc := range testCases {
		extendedTestCases = append(extendedTestCases, takeByTopologyExtendedTestCase{
			tc.description,
			tc.topo,
			tc.availableCPUs,
			tc.numCPUs,
			1,
			tc.expErr,
			tc.expResult,
		})
	}

	extendedTestCases = append(extendedTestCases, []takeByTopologyExtendedTestCase{
		{
			"allocate 4 full cores with 2 distributed across each NUMA node",
			topoDualSocketHT,
			mustParseCPUSet(t, "0-11"),
			8,
			1,
			"",
			mustParseCPUSet(t, "0,6,2,8,1,7,3,9"),
		},
		{
			"allocate 32 full cores with 8 distributed across each NUMA node",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			64,
			1,
			"",
			mustParseCPUSet(t, "0-7,10-17,20-27,30-37,40-47,50-57,60-67,70-77"),
		},
		{
			"allocate 24 full cores with 8 distributed across the first 3 NUMA nodes",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			48,
			1,
			"",
			mustParseCPUSet(t, "0-7,10-17,20-27,40-47,50-57,60-67"),
		},
		{
			"allocate 24 full cores with 8 distributed across the first 3 NUMA nodes (taking all but 2 from the first NUMA node)",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "1-29,32-39,41-69,72-79"),
			48,
			1,
			"",
			mustParseCPUSet(t, "1-8,10-17,20-27,41-48,50-57,60-67"),
		},
		{
			"allocate 24 full cores with 8 distributed across the last 3 NUMA nodes (even though all 8 could be allocated from the first NUMA node)",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "2-29,31-39,42-69,71-79"),
			48,
			1,
			"",
			mustParseCPUSet(t, "10-17,20-27,31-38,50-57,60-67,71-78"),
		},
		{
			"allocate 8 full cores with 2 distributed across each NUMA node",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-2,10-12,20-22,30-32,40-41,50-51,60-61,70-71"),
			16,
			1,
			"",
			mustParseCPUSet(t, "0-1,10-11,20-21,30-31,40-41,50-51,60-61,70-71"),
		},
		{
			"allocate 8 full cores with 2 distributed across each NUMA node",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-2,10-12,20-22,30-32,40-41,50-51,60-61,70-71"),
			16,
			1,
			"",
			mustParseCPUSet(t, "0-1,10-11,20-21,30-31,40-41,50-51,60-61,70-71"),
		},
	}...)

	return extendedTestCases
}

func TestTakeByTopologyNUMADistributed(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := commonTakeByTopologyExtendedTestCases(t)
	testCases = append(testCases, []takeByTopologyExtendedTestCase{
		{
			"take one cpu from dual socket with HT - core from Socket 0",
			topoDualSocketHT,
			cpuset.New(1, 2, 3, 4, 5, 7, 8, 9, 10, 11),
			1,
			1,
			"",
			cpuset.New(1),
		},
		{
			"take one cpu from dual socket with HT - core from Socket 0 - cpuGroupSize 2",
			topoDualSocketHT,
			cpuset.New(1, 2, 3, 4, 5, 7, 8, 9, 10, 11),
			1,
			2,
			"",
			cpuset.New(2),
		},
		{
			"allocate 13 full cores distributed across the first 2 NUMA nodes",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			26,
			1,
			"",
			mustParseCPUSet(t, "0-6,10-16,40-45,50-55"),
		},
		{
			"allocate 13 full cores distributed across the first 2 NUMA nodes (cpuGroupSize 2)",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			26,
			2,
			"",
			mustParseCPUSet(t, "0-6,10-15,40-46,50-55"),
		},
		{
			"allocate 31 full cores with 15 CPUs distributed across each NUMA node and 1 CPU spilling over to each of NUMA 0, 1",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			62,
			1,
			"",
			mustParseCPUSet(t, "0-7,10-17,20-27,30-37,40-47,50-57,60-66,70-76"),
		},
		{
			"allocate 31 full cores with 14 CPUs distributed across each NUMA node and 2 CPUs spilling over to each of NUMA 0, 1, 2 (cpuGroupSize 2)",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-79"),
			62,
			2,
			"",
			mustParseCPUSet(t, "0-7,10-17,20-27,30-36,40-47,50-57,60-67,70-76"),
		},
		{
			"allocate 31 full cores with 15 CPUs distributed across each NUMA node and 1 CPU spilling over to each of NUMA 2, 3 (to keep balance)",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-8,10-18,20-39,40-48,50-58,60-79"),
			62,
			1,
			"",
			mustParseCPUSet(t, "0-7,10-17,20-27,30-37,40-46,50-56,60-67,70-77"),
		},
		{
			"allocate 31 full cores with 14 CPUs distributed across each NUMA node and 2 CPUs spilling over to each of NUMA 0, 2, 3 (to keep balance with cpuGroupSize 2)",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-8,10-18,20-39,40-48,50-58,60-79"),
			62,
			2,
			"",
			mustParseCPUSet(t, "0-7,10-16,20-27,30-37,40-47,50-56,60-67,70-77"),
		},
		{
			"ensure bestRemainder chosen with NUMA nodes that have enough CPUs to satisfy the request",
			topoDualSocketMultiNumaPerSocketHT,
			mustParseCPUSet(t, "0-3,10-13,20-23,30-36,40-43,50-53,60-63,70-76"),
			34,
			1,
			"",
			mustParseCPUSet(t, "0-3,10-13,20-23,30-34,40-43,50-53,60-63,70-74"),
		},
		{
			"ensure previous failure encountered on live machine has been fixed (1/1)",
			topoDualSocketMultiNumaPerSocketHTLarge,
			mustParseCPUSet(t, "0,128,30,31,158,159,43-47,171-175,62,63,190,191,75-79,203-207,94,96,222,223,101-111,229-239,126,127,254,255"),
			28,
			1,
			"",
			mustParseCPUSet(t, "43-47,75-79,96,101-105,171-174,203-206,229-232"),
		},
	}...)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := takeByTopologyNUMADistributed(logger, tc.topo, tc.availableCPUs, tc.numCPUs, tc.cpuGroupSize, CPUSortingStrategyPacked, false)
			if err != nil {
				if tc.expErr == "" {
					t.Errorf("unexpected error [%v]", err)
				}
				if tc.expErr != "" && err.Error() != tc.expErr {
					t.Errorf("expected error to be [%v] but it was [%v]", tc.expErr, err)
				}
				return
			}
			if !result.Equals(tc.expResult) {
				t.Errorf("expected result [%s] to equal [%s]", result, tc.expResult)
			}
		})
	}
}

type takeByTopologyTestCaseForResize struct {
	description   string
	topo          *topology.CPUTopology
	opts          StaticPolicyOptions
	availableCPUs cpuset.CPUSet
	reusableCPUs  cpuset.CPUSet
	mustKeepCPUs  cpuset.CPUSet
	numCPUs       int
	expErr        string
	expResult     cpuset.CPUSet
}

func commonTakeByTopologyTestCasesForResize(t *testing.T) []takeByTopologyTestCaseForResize {
	return []takeByTopologyTestCaseForResize{
		{
			description:   "take more cpus than are available from single socket with HT",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: cpuset.New(0, 2, 4, 6),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0),
			numCPUs:       5,
			expErr:        "not enough cpus available to satisfy request: requested=5, available=4",
			expResult:     cpuset.New(),
		},
		{
			description:   "mustKeepCPUs not a subset of reusableCPUs",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: cpuset.New(0, 2, 4, 6),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0, 2),
			numCPUs:       3,
			expErr:        "requested CPUs to be retained 0,2 are not a subset of reusable CPUs 0",
			expResult:     cpuset.New(),
		},
		{
			description:   "Allocated 1 CPUs, and take 1 cpus from single socket with HT",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0),
			numCPUs:       1,
			expErr:        "",
			expResult:     cpuset.New(0),
		},
		{
			description:   "Allocated 1 CPU, and take 2 cpu from single socket with HT",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0),
			numCPUs:       2,
			expErr:        "",
			expResult:     cpuset.New(0, 4),
		},
		{
			description:   "Allocated 1 CPU, and take 2 cpu from single socket with HT, some cpus are taken, no sibling CPU of allocated CPU",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0,1,3,5,6,7"),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0),
			numCPUs:       2,
			expErr:        "",
			expResult:     cpuset.New(0, 6),
		},
		{
			description:   "Allocated 1 CPU, and take 3 cpu from single socket with HT, some cpus are taken, no sibling CPU of allocated CPU",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0,1,3,5,6,7"),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0),
			numCPUs:       3,
			expErr:        "",
			expResult:     cpuset.New(0, 1, 5),
		},
		{
			description:   "Allocated 1 CPU, and take all cpu from single socket with HT",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0,1-7"),
			reusableCPUs:  cpuset.New(0),
			mustKeepCPUs:  cpuset.New(0),
			numCPUs:       8,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7"),
		},
		{
			description:   "Allocated 1 CPU, take a core from dual socket with HT",
			topo:          topoDualSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       2,
			expErr:        "",
			expResult:     cpuset.New(5, 11),
		},
		{
			description:   "Allocated 1 CPU, take a socket of cpus from dual socket with HT",
			topo:          topoDualSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       6,
			expErr:        "",
			expResult:     cpuset.New(1, 3, 5, 7, 9, 11),
		},
		{
			description:   "Allocated 1 CPU, take a socket of cpus and 1 core of CPU from dual socket with HT",
			topo:          topoDualSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       8,
			expErr:        "",
			expResult:     cpuset.New(0, 1, 3, 5, 6, 7, 9, 11),
		},
		{
			description:   "Allocated 1 CPU, take a socket of cpus from dual socket with multi-numa-per-socket with HT",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  cpuset.New(39),
			mustKeepCPUs:  cpuset.New(39),
			numCPUs:       40,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "20-39,60-79"),
		},
		{
			description:   "Allocated 1 CPU, take a NUMA node of cpus from dual socket with multi-numa-per-socket with HT",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  cpuset.New(39),
			mustKeepCPUs:  cpuset.New(39),
			numCPUs:       20,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "30-39,70-79"),
		},
		{
			description:   "Allocated 2 CPUs, take a socket and a NUMA node of cpus from dual socket with multi-numa-per-socket with HT",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  cpuset.New(39, 59),
			mustKeepCPUs:  cpuset.New(39, 59),
			numCPUs:       60,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-19,30-59,70-79"),
		},
		{
			description:   "Allocated 1 CPU, take NUMA nodes of cpus from dual socket with multi-numa-per-socket with HT, the NUMA node with allocated CPUs already taken some CPUs",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-69"),
			reusableCPUs:  cpuset.New(39),
			mustKeepCPUs:  cpuset.New(39),
			numCPUs:       40,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-8,20-30,39-48,60-69"),
		},
		{
			description:   "Take NUMA nodes of cpus from dual socket with multi-numa-per-socket with HT, the NUMA node with allocated CPUs already taken more CPUs",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "9,30-38,49"),
			reusableCPUs:  cpuset.New(),
			mustKeepCPUs:  cpuset.New(),
			numCPUs:       1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "9"),
		},
		{
			description:   "Allocated 1 CPU, take NUMA nodes of cpus and 1 CPU from dual socket with multi-numa-per-socket with HT, the NUMA node with allocated CPUs already taken some CPUs",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{},
			availableCPUs: mustParseCPUSet(t, "0-69"),
			reusableCPUs:  cpuset.New(39),
			mustKeepCPUs:  cpuset.New(39),
			numCPUs:       41,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-19,39-59"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from single socket with HT, 3 cpus",
			topo:          topoSingleSocketHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  cpuset.New(7),
			mustKeepCPUs:  cpuset.New(7),
			numCPUs:       3,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0,1,7"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from dual socket with HT, 3 cpus",
			topo:          topoDualSocketHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       3,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "1,3,11"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from dual socket with HT, 6 cpus",
			topo:          topoDualSocketHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       6,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "1,3,5,7,9,11"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from dual socket with HT, 8 cpus",
			topo:          topoDualSocketHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       8,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0,1,2,3,5,7,9,11"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from dual socket without HT, 2 cpus",
			topo:          topoDualSocketNoHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  cpuset.New(7),
			mustKeepCPUs:  cpuset.New(7),
			numCPUs:       2,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "4,7"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from dual socket with multi numa per socket and HT, 8 cpus",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  cpuset.New(39),
			mustKeepCPUs:  cpuset.New(39),
			numCPUs:       8,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "20-26,39"),
		},
		{
			description:   "Allocated 1 CPU, take NUMA nodes of cpus from dual socket with multi-numa-per-socket with HT, the NUMA node with allocated CPUs already taken some CPUs",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-69"),
			reusableCPUs:  cpuset.New(39),
			mustKeepCPUs:  cpuset.New(39),
			numCPUs:       40,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-9,20-39,60-69"),
		},
		{
			description:   "Allocated 1 CPUs, take a socket of cpus from quad socket four way with HT, 12 cpus",
			topo:          topoQuadSocketFourWayHT,
			opts:          StaticPolicyOptions{DistributeCPUsAcrossCores: true},
			availableCPUs: mustParseCPUSet(t, "0-287"),
			reusableCPUs:  cpuset.New(60),
			mustKeepCPUs:  cpuset.New(60),
			numCPUs:       8,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "3,4,11,12,15,16,23,60"),
		},
		{
			description:   "Allocated 2 CPUs, take cpus from best available UncoreCache group of multi uncore cache single socket - SMT disabled",
			topo:          topoUncoreSingleSocketNoSMT,
			opts:          StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			availableCPUs: mustParseCPUSet(t, "2-3,8-11,4-7,12-15"),
			reusableCPUs:  cpuset.New(8, 9),
			mustKeepCPUs:  cpuset.New(8, 9),
			numCPUs:       8,
			expErr:        "",
			expResult:     cpuset.New(4, 5, 6, 7, 8, 9, 10, 11),
		},
		{
			description:   "Allocated 2 CPUs, take cpus from the allocated cores in partial UncoreCache - SMT enabled",
			topo:          topoUncoreSingleSocketSMT,
			opts:          StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			availableCPUs: mustParseCPUSet(t, "0-15"),
			reusableCPUs:  cpuset.New(4, 5, 6),
			mustKeepCPUs:  cpuset.New(4, 5, 6),
			numCPUs:       8,
			expErr:        "",
			expResult:     cpuset.New(4, 5, 6, 7, 12, 13, 14, 15),
		},
		{
			description:   "Allocated 2 CPUs, take cpus from the allocated cores in partial UncoreCache - SMT enabled (duplicate)",
			topo:          topoUncoreSingleSocketSMT,
			opts:          StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			availableCPUs: mustParseCPUSet(t, "0-15"),
			reusableCPUs:  cpuset.New(4, 5, 6),
			mustKeepCPUs:  cpuset.New(4, 5, 6),
			numCPUs:       8,
			expErr:        "",
			expResult:     cpuset.New(4, 5, 6, 7, 12, 13, 14, 15),
		},
		{
			description:   "Scale down case, musk Keep 3 CPUs, take cpus from the allocated cores in partial UncoreCache - SMT enabled (duplicate)",
			topo:          topoUncoreSingleSocketSMT,
			opts:          StaticPolicyOptions{PreferAlignByUncoreCacheOption: true},
			availableCPUs: mustParseCPUSet(t, "3-7,11-15"),
			reusableCPUs:  mustParseCPUSet(t, "3-7,11-15"),
			mustKeepCPUs:  cpuset.New(4, 5, 6),
			numCPUs:       8,
			expErr:        "",
			expResult:     cpuset.New(4, 5, 6, 7, 12, 13, 14, 15),
		},
	}
}

func TestTakeByTopologyNUMAPackedForResize(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := commonTakeByTopologyTestCasesForResize(t)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			strategy := CPUSortingStrategyPacked
			if tc.opts.DistributeCPUsAcrossCores {
				strategy = CPUSortingStrategySpread
			}

			result, err := takeByTopologyNUMAPackedForResize(logger, tc.topo, tc.availableCPUs, tc.numCPUs, strategy, tc.opts.PreferAlignByUncoreCacheOption, &tc.reusableCPUs, &tc.mustKeepCPUs)

			if tc.expErr != "" && err != nil && err.Error() != tc.expErr {
				t.Errorf("expected error to be [%v] but it was [%v]", tc.expErr, err)
			}
			if !result.Equals(tc.expResult) {
				t.Errorf("expected result [%s] to equal [%s]", result, tc.expResult)
			}
		})
	}
}

type takeByTopologyExtendedTestCaseForResize struct {
	description   string
	topo          *topology.CPUTopology
	availableCPUs cpuset.CPUSet
	reusableCPUs  cpuset.CPUSet
	mustKeepCPUs  cpuset.CPUSet
	numCPUs       int
	cpuGroupSize  int
	expErr        string
	expResult     cpuset.CPUSet
}

func commonTakeByTopologyExtendedTestCasesForResize(t *testing.T) []takeByTopologyExtendedTestCaseForResize {
	return []takeByTopologyExtendedTestCaseForResize{
		{
			description:   "take more cpus than are available from Dual socket with HT",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  mustParseCPUSet(t, "0,2,4"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,2,4"),
			numCPUs:       10,
			cpuGroupSize:  1,
			expErr:        "not enough cpus available to satisfy request: requested=10, available=8",
			expResult:     cpuset.New(),
		},
		{
			description:   "mustKeepCPUs not a subset of reusableCPUs",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  mustParseCPUSet(t, "0,2,4"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,2,4,6"),
			numCPUs:       5,
			cpuGroupSize:  1,
			expErr:        "requested CPUs to be retained 0,2,4,6 are not a subset of reusable CPUs 0,2,4",
			expResult:     cpuset.New(),
		},
		{
			description:   "Allocated 1 CPUs, allocate 4 full cores with 2 distributed across each NUMA node",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  cpuset.New(11),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       8,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0,6,2,8,1,7,5,11"),
		},
		{
			description:   "Allocated 8 CPUs, allocate 32 full cores with 8 distributed across each NUMA node",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "36-39,76-79"),
			mustKeepCPUs:  mustParseCPUSet(t, "36-39,76-79"),
			numCPUs:       64,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7,10-17,20-27,30-33,36-39,40-47,50-57,60-67,70-73,76-79"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 8 full cores with 2 distributed across each NUMA node",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-2,10-12,20-22,30-32,40-41,50-51,60-61,70-71"),
			reusableCPUs:  mustParseCPUSet(t, "0,1"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,1"),
			numCPUs:       16,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-1,10-11,20-21,30-31,40-41,50-51,60-61,70-71"),
		},
		{
			description:   "Allocated 1 CPUs, take 1 cpu from dual socket with HT - core from Socket 0",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  mustParseCPUSet(t, "11"),
			mustKeepCPUs:  mustParseCPUSet(t, "11"),
			numCPUs:       1,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "11"),
		},
		{
			description:   "Allocated 1 CPUs, take 2 cpu from dual socket with HT - core from Socket 0",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  mustParseCPUSet(t, "11"),
			mustKeepCPUs:  mustParseCPUSet(t, "11"),
			numCPUs:       2,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "5,11"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 31 full cores with 15 CPUs distributed across each NUMA node and 1 CPU spilling over to each of NUMA 0, 1",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "0,1"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,1"),
			numCPUs:       62,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7,10-17,20-27,30-37,40-47,50-57,60-66,70-76"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 31 full cores with 14 CPUs distributed across each NUMA node and 2 CPUs spilling over to each of NUMA 0, 1, 2 (cpuGroupSize 2)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "0,1"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,1"),
			numCPUs:       62,
			cpuGroupSize:  2,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7,10-17,20-27,30-36,40-47,50-57,60-67,70-76"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 31 full cores with 15 CPUs distributed across each NUMA node and 1 CPU spilling over to each of NUMA 2, 3 (to keep balance)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-8,10-18,20-39,40-48,50-58,60-79"),
			reusableCPUs:  mustParseCPUSet(t, "0,1"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,1"),
			numCPUs:       62,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7,10-17,20-27,30-37,40-46,50-56,60-67,70-77"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 31 full cores with 14 CPUs distributed across each NUMA node and 2 CPUs spilling over to each of NUMA 0, 2, 3 (to keep balance with cpuGroupSize 2)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-8,10-18,20-39,40-48,50-58,60-79"),
			reusableCPUs:  mustParseCPUSet(t, "0,1"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,1"),
			numCPUs:       62,
			cpuGroupSize:  2,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7,10-16,20-27,30-37,40-47,50-56,60-67,70-77"),
		},
		{
			description:   "Allocated 4 CPUs, ensure bestRemainder chosen with NUMA nodes that have enough CPUs to satisfy the request",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-3,10-13,20-23,30-36,40-43,50-53,60-63,70-76"),
			reusableCPUs:  mustParseCPUSet(t, "0-3"),
			mustKeepCPUs:  mustParseCPUSet(t, "0-3"),
			numCPUs:       34,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-3,10-13,20-23,30-34,40-43,50-53,60-63,70-74"),
		},
		{
			description:   "Allocated 4 CPUs, ensure previous failure encountered on live machine has been fixed (1/1)",
			topo:          topoDualSocketMultiNumaPerSocketHTLarge,
			availableCPUs: mustParseCPUSet(t, "0,128,30,31,158,159,43-47,171-175,62,63,190,191,75-79,203-207,94,96,222,223,101-111,229-239,126,127,254,255"),
			reusableCPUs:  mustParseCPUSet(t, "43-46"),
			mustKeepCPUs:  mustParseCPUSet(t, "43-46"),
			numCPUs:       28,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "43-47,75-79,96,101-105,171-174,203-206,229-232"),
		},
		{
			description:   "Allocated 14 CPUs, allocate 24 full cores with 8 distributed across the first 3 NUMA nodes",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "0-7,40-47"),
			mustKeepCPUs:  mustParseCPUSet(t, "0-7,40-47"),
			numCPUs:       48,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-7,10-17,20-27,40-47,50-57,60-67"),
		},
		{
			description:   "Allocated 20 CPUs, allocated CPUs in numa0 is bigger than distribute CPUs, allocated CPUs by takeByTopologyNUMAPacked",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "0-9,40-49"),
			mustKeepCPUs:  mustParseCPUSet(t, "0-9,40-49"),
			numCPUs:       48,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-23,40-63"),
		},
		{
			description:   "Allocated 12 CPUs, allocate 24 full cores with 8 distributed across the first 3 NUMA nodes (taking all but 2 from the first NUMA node)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "1-29,32-39,41-69,72-79"),
			reusableCPUs:  mustParseCPUSet(t, "1-7,41-47"),
			mustKeepCPUs:  mustParseCPUSet(t, "1-7,41-47"),
			numCPUs:       48,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "1-8,10-17,20-27,41-48,50-57,60-67"),
		},
		{
			description:   "Allocated 10 CPUs, allocate 24 full cores with 8 distributed across the first 3 NUMA nodes (even though all 8 could be allocated from the first NUMA node)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "2-29,31-39,42-69,71-79"),
			reusableCPUs:  mustParseCPUSet(t, "2-7,42-47"),
			mustKeepCPUs:  mustParseCPUSet(t, "2-7,42-47"),
			numCPUs:       48,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "2-9,10-17,20-27,42-49,50-57,60-67"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 13 full cores distributed across the 2 NUMA nodes",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "30,70"),
			mustKeepCPUs:  mustParseCPUSet(t, "30,70"),
			numCPUs:       26,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "20-26,30-36,60-65,70-75"),
		},
		{
			description:   "Allocated 2 CPUs, allocate 13 full cores distributed across the 2 NUMA nodes (cpuGroupSize 2)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "30,70"),
			mustKeepCPUs:  mustParseCPUSet(t, "30,70"),
			numCPUs:       26,
			cpuGroupSize:  2,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "20-25,30-36,60-65,70-76"),
		},
		{
			description:   "Allocated 3 CPUs, allocated CPUs greater than distribution, allocate 5 CPUs distributed across the 2 NUMA nodes (cpuGroupSize 1)",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-7"),
			reusableCPUs:  mustParseCPUSet(t, "0,2,4"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,2,4"),
			numCPUs:       5,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0,2,4,1,7"),
		},
		{
			description:   "Allocated 17 CPUs, allocated CPUs greater than distribution, allocate 67 CPUs distributed across the 4 NUMA nodes (cpuGroupSize 1)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "0-8,40-47"),
			mustKeepCPUs:  mustParseCPUSet(t, "0-8,40-47"),
			numCPUs:       67,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-8,40-47,10-18,50-57,20-28,60-67,30-37,70-77"),
		},
		{
			description:   "Allocated 18 CPUs, allocated CPUs greater than distribution, allocate 67 CPUs distributed across the 4 NUMA nodes (cpuGroupSize 2)",
			topo:          topoDualSocketMultiNumaPerSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-79"),
			reusableCPUs:  mustParseCPUSet(t, "0-8,40-48"),
			mustKeepCPUs:  mustParseCPUSet(t, "0-8,40-48"),
			numCPUs:       70,
			cpuGroupSize:  2,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0-8,40-48,10-18,50-58,20-28,60-68,30-37,70-77"),
		},
		{
			description:   "Scale down case, must keep 1 CPUs, allocate 4 full cores with 2 distributed across each NUMA node",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-11"),
			reusableCPUs:  mustParseCPUSet(t, "0-11"),
			mustKeepCPUs:  cpuset.New(11),
			numCPUs:       8,
			cpuGroupSize:  1,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0,6,2,8,1,7,5,11"),
		},
		{
			description:   "Fallback to takeByTopologyNUMAPackedForResize due to numCPUs is not divisible by cpuGroupSize",
			topo:          topoDualSocketHT,
			availableCPUs: mustParseCPUSet(t, "0-8"),
			reusableCPUs:  mustParseCPUSet(t, "0,2,4"),
			mustKeepCPUs:  mustParseCPUSet(t, "0,2,4"),
			numCPUs:       5,
			cpuGroupSize:  2,
			expErr:        "",
			expResult:     mustParseCPUSet(t, "0,2,4,6,8"),
		},
	}
}

func TestTakeByTopologyNUMADistributedForResize(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	testCases := commonTakeByTopologyExtendedTestCasesForResize(t)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {

			result, err := takeByTopologyNUMADistributedForResize(logger, tc.topo, tc.availableCPUs, tc.numCPUs, tc.cpuGroupSize, CPUSortingStrategyPacked, &tc.reusableCPUs, &tc.mustKeepCPUs)
			if err != nil {
				if tc.expErr == "" {
					t.Errorf("unexpected error [%v]", err)
				}
				if tc.expErr != "" && err.Error() != tc.expErr {
					t.Errorf("expected error to be [%v] but it was [%v]", tc.expErr, err)
				}
				return
			}
			if !result.Equals(tc.expResult) {
				t.Errorf("expected result [%s] to equal [%s]", result, tc.expResult)
			}
		})
	}
}

func mustParseCPUSet(t *testing.T, s string) cpuset.CPUSet {
	cpus, err := cpuset.Parse(s)
	if err != nil {
		t.Errorf("parsing %q: %v", s, err)
	}
	return cpus
}
