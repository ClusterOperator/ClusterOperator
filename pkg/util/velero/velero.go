package velero

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"time"
)

const defaultVeleroPath = "/usr/local/bin/velero"

func Get(operate string, args []string) ([]VeleroBackup, error) {
	var backups VeleroBackupList
	operates := []string{"get", operate}
	args = append(operates, args...)
	args = append(args, "-o", "json")

	result, err := ExecCommand(defaultVeleroPath, args)
	if err != nil {
		return backups.Items, err
	}

	var backItem VeleroBackup
	if err := json.Unmarshal(result, &backItem); err != nil {
		return backups.Items, err
	}
	if backItem.Spec != nil {
		backups.Items = append(backups.Items, backItem)
		return backups.Items, err
	}

	if err := json.Unmarshal(result, &backups); err != nil {
		return backups.Items, err
	}
	return backups.Items, nil
}

func GetLogs(name, operate string, args []string) ([]byte, error) {
	logs := []string{operate, "logs", name, "--insecure-skip-tls-verify"}
	args = append(logs, args...)
	return ExecCommand(defaultVeleroPath, args)
}

func GetDescribe(name, operate string, args []string) ([]byte, error) {
	describes := []string{operate, "describe", name}
	args = append(describes, args...)
	return ExecCommand(defaultVeleroPath, args)
}

func Delete(name, operate string, args []string) ([]byte, error) {
	command := "echo y| /usr/local/bin/velero delete " + operate + " " + name
	for _, value := range args {
		command = command + " " + value
	}
	return ExecCommand(command, []string{})
}

func Create(name, operate string, args []string) ([]byte, error) {
	backups := []string{operate, "create", name}
	args = append(backups, args...)
	return ExecCommand(defaultVeleroPath, args)
}

func Restore(backupName string, args []string) ([]byte, error) {
	backups := []string{"restore", "create", "--from-backup", backupName}
	args = append(backups, args...)
	return ExecCommand(defaultVeleroPath, args)
}

func Install(args []string) ([]byte, error) {
	install := []string{"install"}
	args = append(install, args...)
	return ExecCommand(defaultVeleroPath, args)
}

func ExecCommand(command string, args []string) ([]byte, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var cmd *exec.Cmd
	if len(args) == 0 {
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", command)
	} else {
		cmd = exec.CommandContext(ctx, command, args...)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, err
	}
	cmd.Stderr = cmd.Stdout
	if err = cmd.Start(); err != nil {
		return []byte{}, err
	}

	var buffer bytes.Buffer
	done := make(chan bool, 1)
	go func() {
		for {
			out := make([]byte, 1024)
			length, err := stdout.Read(out)
			if err != nil {
				break
			}
			if length > 0 {
				buffer.Write(out[:length])
			}
		}
		done <- true
	}()

	select {
	case <-done:
		if err = cmd.Wait(); err != nil {
			return []byte{}, errors.New(buffer.String())
		}
		return buffer.Bytes(), nil
	case <-time.After(time.Second * 20):
		_ = stdout.Close()
		return []byte("time out"), errors.New("read log time out")
	}
}

type VeleroBackupList struct {
	ApiVersion string         `json:"apiVersion"`
	Items      []VeleroBackup `json:"items"`
}

type VeleroBackup struct {
	Kind     string                 `json:"kind"`
	Metadata map[string]interface{} `json:"metadata"`
	Spec     map[string]interface{} `json:"spec"`
	Status   map[string]interface{} `json:"status"`
}
