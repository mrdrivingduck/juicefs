/*
 * JuiceFS, Copyright 2022 Juicedata, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func GetKernelVersion() (major, minor int) {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		buf := make([]byte, 0, 65) // Utsname.Release [65]int8
		for _, v := range uname.Release {
			if v == 0x00 {
				break
			}
			buf = append(buf, byte(v))
		}
		ps := strings.SplitN(string(buf), ".", 3)
		if len(ps) < 2 {
			return
		}
		if major, err = strconv.Atoi(ps[0]); err != nil {
			return
		}
		minor, _ = strconv.Atoi(ps[1])
	}
	return
}

func GetSysInfo() string {
	var (
		kernel    []byte
		osVersion []byte
		err       error
	)

	kernel, _ = exec.Command("cat", "/proc/version").Output()

	if osVersion, err = exec.Command("lsb_release", "-a").Output(); err != nil {
		osVersion, _ = exec.Command("cat", "/etc/os-release").Output()
	}

	return fmt.Sprintf(`
Kernel: 
%s
OS: 
%s`, kernel, osVersion)
}

func SetIOFlusher() {
	err := unix.Prctl(unix.PR_SET_IO_FLUSHER, 1, 0, 0, 0)
	if errors.Is(err, unix.EPERM) {
		logger.Warn("CAP_SYS_RESOURCE is needed for PR_SET_IO_FLUSHER")
	} else if errors.Is(err, unix.EINVAL) {
		logger.Info("PR_SET_IO_FLUSHER, which is introduced by Linux 5.6, is not supported by the running kernel")
	}
}

// Disable transparent huge page
func DisableTHP() {
	for {
		err := unix.Prctl(unix.PR_SET_THP_DISABLE, 1, 0, 0, 0)
		if err == nil {
			break
		}

		if errors.Is(err, unix.EINTR) {
			continue
		} else {
			logger.Warnf("Failed to disable transparent huge page: %s", err)
			return
		}
	}
}

// AdjustOOMKiller: change oom_score_adj to avoid OOM-killer
func AdjustOOMKiller(score int) {
	if os.Getuid() != 0 {
		return
	}
	f, err := os.OpenFile("/proc/self/oom_score_adj", os.O_WRONLY, 0666)
	if err != nil {
		if !os.IsNotExist(err) {
			println(err)
		}
		return
	}
	defer f.Close()
	_, err = f.WriteString(strconv.Itoa(score))
	if err != nil {
		println("adjust OOM score:", err)
	}
}
