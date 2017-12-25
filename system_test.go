package system

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-sysinfo/types"
)

type ProcessFeatures struct {
	ProcessInfo    bool
	Environment    bool
	FileDescriptor bool
	CPUTimer       bool
	Memory         bool
}

var expectedProcessFeatures = map[string]ProcessFeatures{
	"darwin": ProcessFeatures{
		ProcessInfo:    true,
		Environment:    true,
		FileDescriptor: false,
		CPUTimer:       true,
		Memory:         true,
	},
	"linux": ProcessFeatures{
		ProcessInfo:    true,
		Environment:    true,
		FileDescriptor: true,
		CPUTimer:       true,
		Memory:         true,
	},
}

func TestProcessFeaturesMatrix(t *testing.T) {
	const GOOS = runtime.GOOS
	var features ProcessFeatures

	process, err := Self()
	if err == types.ErrNotImplemented {
		assert.Nil(t, expectedProcessFeatures[GOOS], "expected to find a ProcessProvider for %v", GOOS)
		logAsJSON(t, features)
		return
	} else if err != nil {
		t.Fatal(err)
	}
	features.ProcessInfo = true

	_, features.Environment = process.(types.Environment)
	_, features.FileDescriptor = process.(types.FileDescriptor)
	_, features.CPUTimer = process.(types.CPUTimer)
	_, features.Memory = process.(types.Memory)

	assert.Equal(t, expectedProcessFeatures[GOOS], features)
	logAsJSON(t, features)
}

func TestSelf(t *testing.T) {
	process, err := Self()
	if err == types.ErrNotImplemented {
		t.Skip("process provider not implemented on", runtime.GOOS)
	}

	if runtime.GOOS == "linux" {
		// Do some dummy work to spend user CPU time.
		var v int
		for i := 0; i < 999999999; i++ {
			v += i * i
		}
	}

	info, err := process.Info()
	if err != nil {
		t.Fatal(err)
	}
	logAsJSON(t, info)
	assert.EqualValues(t, os.Getpid(), info.PID)
	assert.EqualValues(t, os.Getppid(), info.PPID)
	assert.Equal(t, os.Args, info.Args)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, wd, info.CWD)

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, exe, info.Exe)

	if err != nil {
		t.Fatal(err)
	}
	assert.WithinDuration(t, info.StartTime, time.Now(), 10*time.Second)

	if v, ok := process.(types.Environment); ok {
		expectedEnv := map[string]string{}
		for _, keyValue := range os.Environ() {
			parts := strings.SplitN(keyValue, "=", 2)
			if len(parts) != 2 {
				t.Fatal("failed to parse os.Environ()")
			}
			expectedEnv[parts[0]] = parts[1]
		}
		actualEnv, err := v.Environment()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expectedEnv, actualEnv)
		logAsJSON(t, actualEnv)
	}

	if v, ok := process.(types.Memory); ok {
		memInfo := v.Memory()
		assert.NotZero(t, memInfo.Virtual)
		assert.NotZero(t, memInfo.Resident)
		logAsJSON(t, memInfo)
	}

	if v, ok := process.(types.CPUTimer); ok {
		cpuTimes := v.CPUTime()
		assert.NotZero(t, cpuTimes)
		logAsJSON(t, cpuTimes)
	}

	if v, ok := process.(types.FileDescriptor); ok {
		count, err := v.FileDescriptorCount()
		if assert.NoError(t, err) {
			t.Log("file descriptor count:", count)
		}
		fds, err := v.FileDescriptors()
		if assert.NoError(t, err) {
			logAsJSON(t, fds)
		}
	}
}

func TestHost(t *testing.T) {
	host, err := Host()
	if err == types.ErrNotImplemented {
		t.Skip("host provider not implemented on", runtime.GOOS)
	}

	info := host.Info()
	assert.NotZero(t, info)

	data, _ := json.MarshalIndent(info, "", "  ")
	t.Log(string(data))
}

func logAsJSON(t testing.TB, v interface{}) {
	if !testing.Verbose() {
		return
	}
	t.Helper()
	j, _ := json.MarshalIndent(v, "", "  ")
	t.Log(string(j))
}
