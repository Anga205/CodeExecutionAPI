package executor

import (
	"os/exec"
	"reflect"
	"strings"
	"syscall"
)

func buildSandboxCommand(stdin string, ws SandboxWorkspace, cgroupFD int) *exec.Cmd {
	return buildSandboxCommandWithCgroupMode(stdin, ws, cgroupFD, true)
}

func buildSandboxCommandWithoutCgroupFD(stdin string, ws SandboxWorkspace) *exec.Cmd {
	return buildSandboxCommandWithCgroupMode(stdin, ws, 0, false)
}

func buildSandboxCommandWithCgroupMode(stdin string, ws SandboxWorkspace, cgroupFD int, useCgroupFD bool) *exec.Cmd {
	command := ws.RunCommand
	if ws.SandboxInitExecPath != "" {
		command = append([]string{ws.SandboxInitExecPath}, ws.RunCommand...)
	}

	cmd := exec.Command(command[0], command[1:]...)
	sysProcAttr := &syscall.SysProcAttr{
		Cloneflags:                 syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER,
		Setpgid:                    true,
		UseCgroupFD:                useCgroupFD,
		CgroupFD:                   cgroupFD,
		Credential:                 &syscall.Credential{Uid: 0, Gid: 0},
		UidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: 65534, Size: 1}},
		GidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: 65534, Size: 1}},
		GidMappingsEnableSetgroups: false,
	}
	setNoNewPrivsIfSupported(sysProcAttr)
	if ws.UseChroot {
		sysProcAttr.Chroot = ws.Dir
		cmd.Dir = "/"
	} else {
		cmd.Dir = ws.Dir
	}
	cmd.SysProcAttr = sysProcAttr
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	baseEnv := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/",
		"LANG=C.UTF-8",
	}
	cmd.Env = append([]string{}, baseEnv...)
	if len(ws.ExtraEnv) > 0 {
		cmd.Env = append(cmd.Env, ws.ExtraEnv...)
	}
	return cmd
}

func setNoNewPrivsIfSupported(attr *syscall.SysProcAttr) {
	field := reflect.ValueOf(attr).Elem().FieldByName("NoNewPrivs")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(true)
	}
}
