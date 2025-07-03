/*
   Copyright The containerd Authors.

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

package mountutil

import (
	"io/fs"
	"strings"

	mobymount "github.com/moby/sys/mount"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

/*
   Portions from https://github.com/moby/moby/blob/v20.10.5/daemon/oci_linux.go
   Portions from https://github.com/moby/moby/blob/v20.10.5/volume/mounts/linux_parser.go
   Copyright (C) Docker/Moby authors.
   Licensed under the Apache License, Version 2.0
   NOTICE: https://github.com/moby/moby/blob/v20.10.5/NOTICE
*/

const (
	DefaultMountType = "none"

	// DefaultPropagationMode is the default propagation of mounts
	// where user doesn't specify mount propagation explicitly.
	// See also: https://github.com/moby/moby/blob/v20.10.7/volume/mounts/linux_parser.go#L145
	DefaultPropagationMode = "rprivate"
)

// UnprivilegedMountFlags is from https://github.com/moby/moby/blob/v20.10.5/daemon/oci_linux.go#L420-L450
//
// Get the set of mount flags that are set on the mount that contains the given
// path and are locked by CL_UNPRIVILEGED. This is necessary to ensure that
// bind-mounting "with options" will not fail with user namespaces, due to
// kernel restrictions that require user namespace mounts to preserve
// CL_UNPRIVILEGED locked flags.
func UnprivilegedMountFlags(path string) ([]string, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(path, &statfs); err != nil {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: err}
	}

	// The set of keys come from https://github.com/torvalds/linux/blob/v4.13/fs/namespace.c#L1034-L1048.
	unprivilegedFlags := map[uint64]string{
		unix.MS_RDONLY:     "ro",
		unix.MS_NODEV:      "nodev",
		unix.MS_NOEXEC:     "noexec",
		unix.MS_NOSUID:     "nosuid",
		unix.MS_NOATIME:    "noatime",
		unix.MS_RELATIME:   "relatime",
		unix.MS_NODIRATIME: "nodiratime",
	}

	var flags []string
	for mask, flag := range unprivilegedFlags {
		if uint64(statfs.Flags)&mask == mask {
			flags = append(flags, flag)
		}
	}

	return flags, nil
}

func ProcessFlagTmpfs(s string) (*Processed, error) {
	split := strings.SplitN(s, ":", 2)
	dst := split[0]
	options := []string{"noexec", "nosuid", "nodev"}
	if len(split) == 2 {
		raw := append(options, strings.Split(split[1], ",")...)
		var err error
		options, err = mobymount.MergeTmpfsOptions(raw)
		if err != nil {
			return nil, err
		}
	}
	res := &Processed{
		Mount: specs.Mount{
			Type:        "tmpfs",
			Source:      "tmpfs",
			Destination: dst,
			Options:     options,
		},
		Type: Tmpfs,
		Mode: strings.Join(options, ","),
	}
	return res, nil
}
