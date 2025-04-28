package backup

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Helper function to execute a command over SSH
func executeSSHCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)
	if err != nil {
		return fmt.Errorf("command failed: %w\nStdout: %s\nStderr: %s",
			err, stdoutBuf.String(), stderrBuf.String())
	}

	return nil
}

// Helper function to copy a file from remote to local using SCP
func scpFileFromRemote(client *ssh.Client, remoteFilePath, localFilePath string) error {
	// Create a new session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Set up pipes for file transfer
	var stderr bytes.Buffer
	session.Stderr = &stderr

	// Start remote scp command
	// "scp -f" means "send file to client"
	remoteCmd := fmt.Sprintf("cat %s", remoteFilePath)
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	err = session.Start(remoteCmd)
	if err != nil {
		return fmt.Errorf("failed to start remote command: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy data from remote to local
	_, err = io.Copy(localFile, stdout)
	if err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	// Wait for command to complete
	err = session.Wait()
	if err != nil {
		return fmt.Errorf("command failed: %w\nStderr: %s", err, stderr.String())
	}

	return nil
}

// Helper function to copy a file from local to remote using SCP
func scpFileToRemote(client *ssh.Client, localFilePath, remoteFilePath string) error {
	// Open and stat the local file
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// Get file info for permissions and size
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}
	fileSize := fileInfo.Size()

	// Create a new SSH session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Set up pipes for file transfer
	var stderr bytes.Buffer
	session.Stderr = &stderr

	// Get stdin pipe to write file data
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// Escape remote path for shell
	escapedDir := escapeShellArg(filepath.Dir(remoteFilePath))
	escapedPath := escapeShellArg(remoteFilePath)

	// Make sure remote directory exists
	mkdirCmd := fmt.Sprintf("mkdir -p %s", escapedDir)
	err = executeSSHCommand(client, mkdirCmd)
	if err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Start file transfer on the remote side with dd to handle binary data better
	err = session.Start(fmt.Sprintf("dd of=%s bs=32k", escapedPath))
	if err != nil {
		return fmt.Errorf("failed to start remote command: %w", err)
	}

	// Copy file content to remote with progress tracking
	written, err := io.Copy(stdin, localFile)
	if err != nil {
		// Try to remove incomplete file
		cleanupSession, _ := client.NewSession()
		if cleanupSession != nil {
			defer cleanupSession.Close()
			cleanupSession.Run(fmt.Sprintf("rm -f %s", escapedPath))
		}
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	// Check if we wrote the entire file
	if written != fileSize {
		// Try to remove incomplete file
		cleanupSession, _ := client.NewSession()
		if cleanupSession != nil {
			defer cleanupSession.Close()
			cleanupSession.Run(fmt.Sprintf("rm -f %s", escapedPath))
		}
		return fmt.Errorf("incomplete file transfer: wrote %d bytes out of %d", written, fileSize)
	}

	// Close stdin to signal end of file transfer
	err = stdin.Close()
	if err != nil {
		return fmt.Errorf("failed to close stdin pipe: %w", err)
	}

	// Wait for command to complete
	err = session.Wait()
	if err != nil {
		return fmt.Errorf("command failed: %w\nStderr: %s", err, stderr.String())
	}

	// Set file permissions
	chmod := fmt.Sprintf("chmod %o %s", fileInfo.Mode().Perm(), escapedPath)
	err = executeSSHCommand(client, chmod)
	if err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Verify file size
	verifyCmd := fmt.Sprintf("stat -c %%s %s", escapedPath)
	verifySession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create verification session: %w", err)
	}
	defer verifySession.Close()

	var sizeOutput bytes.Buffer
	verifySession.Stdout = &sizeOutput

	if err := verifySession.Run(verifyCmd); err != nil {
		return fmt.Errorf("failed to verify file size: %w", err)
	}

	remoteSizeStr := strings.TrimSpace(sizeOutput.String())
	remoteSize, err := strconv.ParseInt(remoteSizeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse remote file size: %w", err)
	}

	if remoteSize != fileSize {
		return fmt.Errorf("file size mismatch: expected %d bytes, got %d bytes", fileSize, remoteSize)
	}

	return nil
}

// Helper function to escape shell arguments
func escapeShellArg(arg string) string {
	return "'" + strings.Replace(arg, "'", "'\\''", -1) + "'"
}
