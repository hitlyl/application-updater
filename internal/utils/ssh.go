package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig contains SSH connection parameters
type SSHConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Timeout  time.Duration
}

// ExecuteSSHCommand runs a command on a remote host via SSH
func ExecuteSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	err = session.Run(command)
	if err != nil {
		return stdout.String(), err
	}

	return stdout.String(), nil
}

// SCPFileToRemote copies a local file to a remote server via SCP
func SCPFileToRemote(client *ssh.Client, localFilePath, remoteFilePath string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %v", err)
	}

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()

		fmt.Fprintf(w, "C%#o %d %s\n", stat.Mode().Perm(), stat.Size(), filepath.Base(remoteFilePath))
		io.Copy(w, file)
		fmt.Fprint(w, "\x00")
	}()

	cmd := fmt.Sprintf("scp -t %s", filepath.Dir(remoteFilePath))
	err = session.Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to run scp command: %v", err)
	}

	return nil
}

// CreateSSHClient creates a new SSH client with the specified credentials
func CreateSSHClient(host, user, password string, port int) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", addr, err)
	}

	return client, nil
}
